package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Integration Demo: Shows how to bridge the two systems
// This demonstrates the exact gap and solution

func main() {
	fmt.Println("=== FLOW2 INTEGRATION GAP DEMONSTRATION ===")
	
	// CURRENT STATE: Two separate systems
	fmt.Println("\n🔍 CURRENT STATE: Two Parallel Systems")
	
	// System 1: Existing Flow2 Orchestrator Output
	basicOutput := simulateCurrentFlow2Output()
	fmt.Println("\n📊 System 1 - Current Flow2 Orchestrator Output:")
	printJSON(basicOutput)
	
	// System 2: Enhanced Proposal Generator Output  
	enhancedOutput := simulateEnhancedProposalOutput()
	fmt.Println("\n🎯 System 2 - Enhanced Proposal Generator Output:")
	printJSON(enhancedOutput)
	
	// THE GAP: They don't connect
	fmt.Println("\n❌ THE GAP: These systems don't talk to each other!")
	fmt.Println("   • Different entry points")
	fmt.Println("   • Different data flows") 
	fmt.Println("   • Different output formats")
	
	// THE SOLUTION: Bridge them
	fmt.Println("\n✅ THE SOLUTION: Bridge Integration")
	bridgedOutput := demonstrateBridgeIntegration(basicOutput)
	fmt.Println("\n🔗 Bridged Output - Enhanced from Existing Pipeline:")
	printJSON(bridgedOutput)
	
	fmt.Println("\n🎉 INTEGRATION COMPLETE!")
	fmt.Println("   • Uses existing 4-step pipeline data")
	fmt.Println("   • Transforms to comprehensive clinical intelligence")
	fmt.Println("   • No duplication of work")
	fmt.Println("   • Maintains performance")
}

// Current Flow2 basic output structure
type BasicMedicationProposal struct {
	SafetyStatus    string                     `json:"safetyStatus"`
	Recommendations []BasicMedicationRecommendation `json:"recommendations"`
}

type BasicMedicationRecommendation struct {
	MedicationName string  `json:"medicationName"`
	DosageForm     string  `json:"dosageForm"`
	Strength       string  `json:"strength"`
	Route          string  `json:"route"`
	Frequency      string  `json:"frequency"`
	Duration       string  `json:"duration"`
	Instructions   string  `json:"instructions"`
	SafetyScore    float64 `json:"safetyScore"`
}

// Enhanced proposal structure (simplified for demo)
type EnhancedProposal struct {
	ProposalID      string                `json:"proposalId"`
	ProposalVersion string                `json:"proposalVersion"`
	Timestamp       time.Time             `json:"timestamp"`
	Metadata        ProposalMetadata      `json:"metadata"`
	CalculatedOrder CalculatedOrder       `json:"calculatedOrder"`
	MonitoringPlan  MonitoringPlan        `json:"monitoringPlan"`
	ClinicalRationale ClinicalRationale   `json:"clinicalRationale"`
}

type ProposalMetadata struct {
	PatientID       string  `json:"patientId"`
	Status          string  `json:"status"`
	ConfidenceScore float64 `json:"confidenceScore"`
}

type CalculatedOrder struct {
	Medication MedicationDetail `json:"medication"`
	Dosing     DosingDetail     `json:"dosing"`
}

type MedicationDetail struct {
	GenericName      string `json:"genericName"`
	TherapeuticClass string `json:"therapeuticClass"`
}

type DosingDetail struct {
	PatientInstructions string `json:"patientInstructions"`
	SafetyScore        float64 `json:"safetyScore"`
}

type MonitoringPlan struct {
	OverallRisk       string   `json:"overallRisk"`
	BaselineChecks    []string `json:"baselineChecks"`
	OngoingMonitoring []string `json:"ongoingMonitoring"`
}

type ClinicalRationale struct {
	Decision   string `json:"decision"`
	Confidence string `json:"confidence"`
	Rationale  string `json:"rationale"`
}

// Simulate current Flow2 orchestrator output
func simulateCurrentFlow2Output() *BasicMedicationProposal {
	return &BasicMedicationProposal{
		SafetyStatus: "safe",
		Recommendations: []BasicMedicationRecommendation{
			{
				MedicationName: "Metformin",
				DosageForm:     "tablet",
				Strength:       "500mg",
				Route:          "PO",
				Frequency:      "q24h",
				Duration:       "ongoing",
				Instructions:   "Take as directed", // Generic!
				SafetyScore:    0.95,
			},
		},
	}
}

// Simulate enhanced proposal generator output
func simulateEnhancedProposalOutput() *EnhancedProposal {
	return &EnhancedProposal{
		ProposalID:      "prop-enhanced-123",
		ProposalVersion: "1.0",
		Timestamp:       time.Now(),
		Metadata: ProposalMetadata{
			PatientID:       "patient-456",
			Status:          "PROPOSED",
			ConfidenceScore: 0.95,
		},
		CalculatedOrder: CalculatedOrder{
			Medication: MedicationDetail{
				GenericName:      "Metformin",
				TherapeuticClass: "Biguanides",
			},
			Dosing: DosingDetail{
				PatientInstructions: "Take 1 tablet by mouth once daily with breakfast", // Specific!
				SafetyScore:        0.95,
			},
		},
		MonitoringPlan: MonitoringPlan{
			OverallRisk:       "MODERATE",
			BaselineChecks:    []string{"eGFR", "HbA1c", "Vitamin B12"},
			OngoingMonitoring: []string{"eGFR every 12 months", "HbA1c every 3 months"},
		},
		ClinicalRationale: ClinicalRationale{
			Decision:   "Initiate Metformin 500mg daily for newly diagnosed Type 2 Diabetes",
			Confidence: "HIGH",
			Rationale:  "First-line therapy with excellent safety profile and proven efficacy",
		},
	}
}

// THE BRIDGE: Transform basic output to enhanced using existing pipeline data
func demonstrateBridgeIntegration(basicOutput *BasicMedicationProposal) *EnhancedProposal {
	if len(basicOutput.Recommendations) == 0 {
		return nil
	}
	
	// Use the existing pipeline's top recommendation
	topRec := basicOutput.Recommendations[0]
	
	// Transform to enhanced structure using the SAME data
	return &EnhancedProposal{
		ProposalID:      "prop-bridged-789",
		ProposalVersion: "1.0",
		Timestamp:       time.Now(),
		Metadata: ProposalMetadata{
			PatientID:       "patient-456",
			Status:          "PROPOSED",
			ConfidenceScore: topRec.SafetyScore, // Use existing safety score
		},
		CalculatedOrder: CalculatedOrder{
			Medication: MedicationDetail{
				GenericName:      topRec.MedicationName, // Use existing data
				TherapeuticClass: "Biguanides", // Would be enriched from drug database
			},
			Dosing: DosingDetail{
				// ENHANCED: Transform generic to specific instructions
				PatientInstructions: enhanceInstructions(topRec.MedicationName, topRec.Frequency),
				SafetyScore:        topRec.SafetyScore,
			},
		},
		MonitoringPlan: MonitoringPlan{
			// ENHANCED: Add monitoring based on medication
			OverallRisk:       assessRisk(topRec.SafetyScore),
			BaselineChecks:    getBaselineChecks(topRec.MedicationName),
			OngoingMonitoring: getOngoingMonitoring(topRec.MedicationName),
		},
		ClinicalRationale: ClinicalRationale{
			// ENHANCED: Generate clinical rationale
			Decision:   fmt.Sprintf("Recommend %s based on safety analysis", topRec.MedicationName),
			Confidence: getConfidenceLevel(topRec.SafetyScore),
			Rationale:  generateRationale(topRec.MedicationName, topRec.SafetyScore),
		},
	}
}

// Helper functions to demonstrate enhancement
func enhanceInstructions(medication, frequency string) string {
	if medication == "Metformin" {
		return "Take 1 tablet by mouth once daily with breakfast to minimize GI upset"
	}
	return "Take as directed by your healthcare provider"
}

func assessRisk(safetyScore float64) string {
	if safetyScore > 0.9 {
		return "LOW"
	} else if safetyScore > 0.7 {
		return "MODERATE"
	}
	return "HIGH"
}

func getBaselineChecks(medication string) []string {
	if medication == "Metformin" {
		return []string{"eGFR", "HbA1c", "Vitamin B12"}
	}
	return []string{"Basic metabolic panel"}
}

func getOngoingMonitoring(medication string) []string {
	if medication == "Metformin" {
		return []string{"eGFR every 12 months", "HbA1c every 3 months"}
	}
	return []string{"Follow up as clinically indicated"}
}

func getConfidenceLevel(safetyScore float64) string {
	if safetyScore > 0.9 {
		return "HIGH"
	} else if safetyScore > 0.7 {
		return "MODERATE"
	}
	return "LOW"
}

func generateRationale(medication string, safetyScore float64) string {
	return fmt.Sprintf("Selected based on safety analysis (score: %.2f) and clinical appropriateness", safetyScore)
}

func printJSON(v interface{}) {
	jsonData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("Error marshaling: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}
