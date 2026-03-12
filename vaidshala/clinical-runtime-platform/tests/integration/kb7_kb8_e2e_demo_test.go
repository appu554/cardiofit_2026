// Package integration provides end-to-end demonstration tests for KB-7 and KB-8 services.
//
// This test shows clear INPUT → OUTPUT for both:
// - KB-7: Terminology Service (ValueSet expansion, code validation)
// - KB-8: Calculator Service (eGFR, ASCVD, CHA2DS2-VASc, etc.)
package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

const (
	demoKB7URL = "http://localhost:8087"
	demoKB8URL = "http://localhost:8093"
)

func getDemoKB7URL() string {
	if url := os.Getenv("KB7_URL"); url != "" {
		return url
	}
	return demoKB7URL
}

func getDemoKB8URL() string {
	if url := os.Getenv("KB8_URL"); url != "" {
		return url
	}
	return demoKB8URL
}

func printJSON(label string, data interface{}) {
	jsonBytes, _ := json.MarshalIndent(data, "  ", "  ")
	fmt.Printf("\n%s:\n  %s\n", label, string(jsonBytes))
}

func printSeparator(title string) {
	fmt.Printf("\n%s\n%s\n%s\n", strings.Repeat("═", 70), fmt.Sprintf("  %s", title), strings.Repeat("═", 70))
}

// TestKB7TerminologyServiceDemo demonstrates KB-7 Terminology Service capabilities
func TestKB7TerminologyServiceDemo(t *testing.T) {
	kb7URL := getDemoKB7URL()

	// Check if KB-7 is running
	resp, err := http.Get(kb7URL + "/health")
	if err != nil {
		t.Skipf("KB-7 not available: %v", err)
	}
	resp.Body.Close()

	printSeparator("KB-7 TERMINOLOGY SERVICE DEMO")

	// ═══════════════════════════════════════════════════════════════
	// TEST 1: Validate SNOMED Code for Diabetes
	// ═══════════════════════════════════════════════════════════════
	t.Run("validate_diabetes_snomed_code", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 1: Validate SNOMED Code for Diabetes Mellitus            │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"code":   "73211009",
			"system": "http://snomed.info/sct",
		}
		printJSON("📥 INPUT", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb7URL+"/api/v1/codes/validate", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Logf("⚠️  Request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)
		fmt.Printf("\n  ✅ SNOMED Code 73211009 = %v\n", result["display"])
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: Expand ValueSet for Diabetes Medications
	// ═══════════════════════════════════════════════════════════════
	t.Run("expand_diabetes_medications_valueset", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 2: Expand ValueSet - Diabetes Medications                 │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		// Try to get available valuesets first
		resp, err := http.Get(kb7URL + "/api/v1/valuesets")
		if err != nil {
			t.Logf("⚠️  Could not list valuesets: %v", err)
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 AVAILABLE VALUESETS", result)
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 3: Check Code Membership in ValueSet
	// ═══════════════════════════════════════════════════════════════
	t.Run("check_code_membership", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 3: Check if Metformin is in Diabetes Medications          │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"code":       "860975", // RxNorm code for Metformin
			"system":     "http://www.nlm.nih.gov/research/umls/rxnorm",
			"valueSetId": "diabetes-medications",
		}
		printJSON("📥 INPUT", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb7URL+"/api/v1/valuesets/membership", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Logf("⚠️  Request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 4: Resolve LOINC Code
	// ═══════════════════════════════════════════════════════════════
	t.Run("resolve_loinc_code", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 4: Resolve LOINC Code for Serum Creatinine                │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"code":   "2160-0",
			"system": "http://loinc.org",
		}
		printJSON("📥 INPUT", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb7URL+"/api/v1/codes/resolve", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Logf("⚠️  Request failed: %v", err)
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)
	})
}

// TestKB8CalculatorServiceDemo demonstrates KB-8 Calculator Service capabilities
func TestKB8CalculatorServiceDemo(t *testing.T) {
	kb8URL := getDemoKB8URL()

	// Check if KB-8 is running
	resp, err := http.Get(kb8URL + "/health")
	if err != nil {
		t.Skipf("KB-8 not available: %v", err)
	}
	resp.Body.Close()

	printSeparator("KB-8 CALCULATOR SERVICE DEMO")

	// ═══════════════════════════════════════════════════════════════
	// TEST 1: eGFR Calculation (CKD-EPI 2021)
	// ═══════════════════════════════════════════════════════════════
	t.Run("calculate_egfr", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 1: eGFR Calculation (CKD-EPI 2021 Race-Free Equation)     │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"serumCreatinine": 1.4,
			"ageYears":        68,
			"sex":             "female",
		}
		printJSON("📥 INPUT (68-year-old female, Creatinine 1.4 mg/dL)", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb8URL+"/api/v1/calculate/egfr", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)

		// Extract key values
		value := result["value"].(float64)
		stage := result["ckdStage"]
		fmt.Printf("\n  ✅ eGFR = %.1f mL/min/1.73m²\n", value)
		fmt.Printf("  ✅ CKD Stage = %v\n", stage)
		fmt.Printf("  📋 Clinical Meaning: %v\n", result["interpretation"])
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 2: ASCVD 10-Year Risk (Pooled Cohort Equations)
	// ═══════════════════════════════════════════════════════════════
	t.Run("calculate_ascvd", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 2: ASCVD 10-Year Risk (Pooled Cohort Equations 2013)      │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"ageYears":         55,
			"sex":              "male",
			"race":             "white",
			"totalCholesterol": 240,
			"hdlCholesterol":   38,
			"systolicBP":       150,
			"onBPTreatment":    true,
			"hasDiabetes":      true,
			"isSmoker":         false,
		}
		printJSON("📥 INPUT (55M, TC 240, HDL 38, SBP 150, on BP meds, diabetic)", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb8URL+"/api/v1/calculate/ascvd", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)

		riskPercent := result["riskPercent"].(float64)
		category := result["riskCategory"]
		fmt.Printf("\n  ✅ 10-Year ASCVD Risk = %.1f%%\n", riskPercent)
		fmt.Printf("  ✅ Risk Category = %v\n", category)
		fmt.Printf("  📋 Statin Recommendation: %v\n", result["statinRecommendation"])
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 3: CHA₂DS₂-VASc Score (AFib Stroke Risk)
	// ═══════════════════════════════════════════════════════════════
	t.Run("calculate_cha2ds2vasc", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 3: CHA₂DS₂-VASc Score (AFib Stroke Risk)                  │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"ageYears":                  78,
			"sex":                       "female",
			"hasCongestiveHeartFailure": true,
			"hasHypertension":           true,
			"hasDiabetes":               true,
			"hasStrokeTIA":              false,
			"hasVascularDisease":        true,
		}
		printJSON("📥 INPUT (78F with CHF, HTN, DM, vascular disease)", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb8URL+"/api/v1/calculate/cha2ds2vasc", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)

		total := result["total"].(float64)
		anticoag := result["anticoagulationRecommended"]
		fmt.Printf("\n  ✅ CHA₂DS₂-VASc Score = %.0f\n", total)
		fmt.Printf("  ✅ Anticoagulation Recommended = %v\n", anticoag)
		fmt.Printf("  📋 Stroke Risk: %v\n", result["annualStrokeRisk"])
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 4: HAS-BLED Score (Bleeding Risk)
	// ═══════════════════════════════════════════════════════════════
	t.Run("calculate_hasbled", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 4: HAS-BLED Score (Major Bleeding Risk)                   │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"hasUncontrolledHypertension": true,
			"hasAbnormalRenalFunction":    true,
			"hasAbnormalLiverFunction":    false,
			"hasStrokeHistory":            false,
			"hasBleedingHistory":          true,
			"hasLabileINR":                false,
			"ageYears":                    72,
			"takingAntiplateletOrNSAID":   true,
			"excessiveAlcohol":            false,
		}
		printJSON("📥 INPUT (72yo with uncontrolled HTN, renal dysfunction, bleeding hx, on antiplatelet)", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb8URL+"/api/v1/calculate/hasbled", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)

		total := result["total"].(float64)
		highRisk := result["highRisk"]
		fmt.Printf("\n  ✅ HAS-BLED Score = %.0f\n", total)
		fmt.Printf("  ✅ High Bleeding Risk = %v\n", highRisk)
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 5: BMI Calculation (with Asian Thresholds)
	// ═══════════════════════════════════════════════════════════════
	t.Run("calculate_bmi", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 5: BMI with Western and Asian (WHO Asia-Pacific) Cutoffs  │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"weightKg": 72,
			"heightCm": 165,
			"region":   "asia",
		}
		printJSON("📥 INPUT (72 kg, 165 cm, Asian patient)", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb8URL+"/api/v1/calculate/bmi", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)

		bmi := result["value"].(float64)
		western := result["categoryWestern"]
		asian := result["categoryAsian"]
		fmt.Printf("\n  ✅ BMI = %.1f kg/m²\n", bmi)
		fmt.Printf("  ✅ Western Category = %v\n", western)
		fmt.Printf("  ✅ Asian Category = %v (lower thresholds for Asian populations)\n", asian)
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 6: SOFA Score (ICU Mortality)
	// ═══════════════════════════════════════════════════════════════
	t.Run("calculate_sofa", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 6: SOFA Score (Sequential Organ Failure Assessment)       │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"pao2fio2Ratio":    150,  // Respiratory: PaO2/FiO2 ratio
			"platelets":        90,   // Coagulation: platelets (×10³/µL)
			"bilirubin":        2.5,  // Liver: bilirubin (mg/dL)
			"map":              60,   // Cardiovascular: MAP (mmHg)
			"glasgowComaScale": 11,   // Neurological: GCS
			"creatinine":       2.8,  // Renal: creatinine (mg/dL)
			"urineOutput":      350,  // Renal: 24h urine output (mL)
		}
		printJSON("📥 INPUT (ICU patient with multi-organ dysfunction)", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb8URL+"/api/v1/calculate/sofa", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)

		total := result["total"].(float64)
		riskLevel := result["riskLevel"]
		fmt.Printf("\n  ✅ SOFA Score = %.0f\n", total)
		fmt.Printf("  ✅ Risk Level = %v\n", riskLevel)

		// Show component scores if available
		if components, ok := result["componentScores"].(map[string]interface{}); ok {
			fmt.Println("  📊 Component Scores:")
			for organ, score := range components {
				fmt.Printf("      - %s: %.0f\n", organ, score)
			}
		}
	})

	// ═══════════════════════════════════════════════════════════════
	// TEST 7: qSOFA Score (Bedside Sepsis Screening)
	// ═══════════════════════════════════════════════════════════════
	t.Run("calculate_qsofa", func(t *testing.T) {
		fmt.Println("\n┌─────────────────────────────────────────────────────────────────┐")
		fmt.Println("│  TEST 7: qSOFA Score (Quick Sepsis Screening - No Labs Needed)  │")
		fmt.Println("└─────────────────────────────────────────────────────────────────┘")

		input := map[string]interface{}{
			"respiratoryRate":  24,    // breaths/min (≥22 = 1 point)
			"systolicBP":       95,    // mmHg (≤100 = 1 point)
			"alteredMentation": true,  // GCS < 15 (= 1 point)
		}
		printJSON("📥 INPUT (RR 24, SBP 95, altered mental status)", input)

		body, _ := json.Marshal(input)
		resp, err := http.Post(kb8URL+"/api/v1/calculate/qsofa", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		json.Unmarshal(respBody, &result)

		printJSON("📤 OUTPUT", result)

		total := result["total"].(float64)
		positive := result["positive"]
		fmt.Printf("\n  ✅ qSOFA Score = %.0f\n", total)
		fmt.Printf("  ✅ Sepsis Screen Positive = %v (≥2 = positive)\n", positive)
		if positive == true {
			fmt.Println("  ⚠️  ALERT: Consider full sepsis workup and SOFA calculation!")
		}
	})
}

// TestKB7AndKB8CombinedWorkflow demonstrates how KB-7 and KB-8 work together
func TestKB7AndKB8CombinedWorkflow(t *testing.T) {
	kb7URL := getDemoKB7URL()
	kb8URL := getDemoKB8URL()

	// Check both services
	resp7, err := http.Get(kb7URL + "/health")
	if err != nil {
		t.Skipf("KB-7 not available: %v", err)
	}
	resp7.Body.Close()

	resp8, err := http.Get(kb8URL + "/health")
	if err != nil {
		t.Skipf("KB-8 not available: %v", err)
	}
	resp8.Body.Close()

	printSeparator("KB-7 + KB-8 COMBINED WORKFLOW DEMO")

	fmt.Print(`
┌─────────────────────────────────────────────────────────────────────────────┐
│                      CLINICAL DECISION SUPPORT FLOW                         │
│                                                                             │
│   FHIR Bundle ──▶ KB-2A (Data Assembly)                                     │
│                        │                                                    │
│                        ▼                                                    │
│              KnowledgeSnapshotBuilder                                       │
│                   │           │                                             │
│                   ▼           ▼                                             │
│              ┌────────┐  ┌────────┐                                         │
│              │  KB-7  │  │  KB-8  │                                         │
│              │ Termin │  │ Calcs  │                                         │
│              └────────┘  └────────┘                                         │
│                   │           │                                             │
│                   ▼           ▼                                             │
│              KnowledgeSnapshot (enriched)                                   │
│                        │                                                    │
│                        ▼                                                    │
│              ClinicalExecutionContext                                       │
│                        │                                                    │
│                        ▼                                                    │
│                   Rule Engine                                               │
└─────────────────────────────────────────────────────────────────────────────┘
`)

	t.Run("combined_patient_workflow", func(t *testing.T) {
		fmt.Println("\n╔═════════════════════════════════════════════════════════════════╗")
		fmt.Println("║  SCENARIO: 65-year-old diabetic patient with AFib               ║")
		fmt.Println("╚═════════════════════════════════════════════════════════════════╝")

		patientData := map[string]interface{}{
			"demographics": map[string]interface{}{
				"age":    65,
				"sex":    "male",
				"weight": 85,
				"height": 170,
			},
			"labs": map[string]interface{}{
				"creatinine":       1.6,
				"totalCholesterol": 210,
				"hdlCholesterol":   42,
			},
			"vitals": map[string]interface{}{
				"systolicBP": 142,
			},
			"conditions": []string{
				"Diabetes mellitus (SNOMED: 73211009)",
				"Atrial fibrillation (SNOMED: 49436004)",
				"Hypertension (SNOMED: 38341003)",
			},
		}

		fmt.Println("\n📋 PATIENT DATA:")
		printJSON("Patient Context", patientData)

		// Step 1: KB-8 calculates eGFR
		fmt.Println("\n─── STEP 1: KB-8 calculates eGFR ───")
		egfrInput := map[string]interface{}{
			"serumCreatinine": 1.6,
			"ageYears":        65,
			"sex":             "male",
		}
		body, _ := json.Marshal(egfrInput)
		resp, _ := http.Post(kb8URL+"/api/v1/calculate/egfr", "application/json", bytes.NewReader(body))
		respBody, _ := io.ReadAll(resp.Body)
		var egfrResult map[string]interface{}
		json.Unmarshal(respBody, &egfrResult)
		resp.Body.Close()

		fmt.Printf("  INPUT:  Creatinine=1.6, Age=65, Sex=male\n")
		fmt.Printf("  OUTPUT: eGFR=%.1f mL/min/1.73m², Stage=%v\n", egfrResult["value"], egfrResult["ckdStage"])

		// Step 2: KB-8 calculates ASCVD risk
		fmt.Println("\n─── STEP 2: KB-8 calculates ASCVD 10-year risk ───")
		ascvdInput := map[string]interface{}{
			"ageYears":         65,
			"sex":              "male",
			"race":             "white",
			"totalCholesterol": 210,
			"hdlCholesterol":   42,
			"systolicBP":       142,
			"onBPTreatment":    true,
			"hasDiabetes":      true,
			"isSmoker":         false,
		}
		body, _ = json.Marshal(ascvdInput)
		resp, _ = http.Post(kb8URL+"/api/v1/calculate/ascvd", "application/json", bytes.NewReader(body))
		respBody, _ = io.ReadAll(resp.Body)
		var ascvdResult map[string]interface{}
		json.Unmarshal(respBody, &ascvdResult)
		resp.Body.Close()

		fmt.Printf("  INPUT:  TC=210, HDL=42, SBP=142, DM=true, BP-treated=true\n")
		fmt.Printf("  OUTPUT: ASCVD Risk=%.1f%%, Category=%v\n", ascvdResult["riskPercent"], ascvdResult["riskCategory"])

		// Step 3: KB-8 calculates CHA2DS2-VASc for AFib
		fmt.Println("\n─── STEP 3: KB-8 calculates CHA₂DS₂-VASc (AFib stroke risk) ───")
		cha2ds2Input := map[string]interface{}{
			"ageYears":                  65,
			"sex":                       "male",
			"hasCongestiveHeartFailure": false,
			"hasHypertension":           true,
			"hasDiabetes":               true,
			"hasStrokeTIA":              false,
			"hasVascularDisease":        false,
		}
		body, _ = json.Marshal(cha2ds2Input)
		resp, _ = http.Post(kb8URL+"/api/v1/calculate/cha2ds2vasc", "application/json", bytes.NewReader(body))
		respBody, _ = io.ReadAll(resp.Body)
		var cha2ds2Result map[string]interface{}
		json.Unmarshal(respBody, &cha2ds2Result)
		resp.Body.Close()

		fmt.Printf("  INPUT:  Age=65, HTN=true, DM=true, Male\n")
		fmt.Printf("  OUTPUT: Score=%.0f, Anticoag Recommended=%v\n", cha2ds2Result["total"], cha2ds2Result["anticoagulationRecommended"])

		// Step 4: KB-8 calculates HAS-BLED
		fmt.Println("\n─── STEP 4: KB-8 calculates HAS-BLED (bleeding risk) ───")
		hasBledInput := map[string]interface{}{
			"hasUncontrolledHypertension": false,
			"hasAbnormalRenalFunction":    true, // eGFR < 60
			"hasAbnormalLiverFunction":    false,
			"hasStrokeHistory":            false,
			"hasBleedingHistory":          false,
			"hasLabileINR":                false,
			"ageYears":                    65,
			"takingAntiplateletOrNSAID":   false,
			"excessiveAlcohol":            false,
		}
		body, _ = json.Marshal(hasBledInput)
		resp, _ = http.Post(kb8URL+"/api/v1/calculate/hasbled", "application/json", bytes.NewReader(body))
		respBody, _ = io.ReadAll(resp.Body)
		var hasBledResult map[string]interface{}
		json.Unmarshal(respBody, &hasBledResult)
		resp.Body.Close()

		fmt.Printf("  INPUT:  Age=65, Renal dysfunction=true (eGFR<60)\n")
		fmt.Printf("  OUTPUT: Score=%.0f, High Risk=%v\n", hasBledResult["total"], hasBledResult["highRisk"])

		// Summary
		fmt.Println("\n╔═════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                    CLINICAL SUMMARY                             ║")
		fmt.Println("╠═════════════════════════════════════════════════════════════════╣")
		fmt.Printf("║  eGFR: %.1f mL/min/1.73m² → CKD Stage %v                       ║\n", egfrResult["value"], egfrResult["ckdStage"])
		fmt.Printf("║  ASCVD 10-Year Risk: %.1f%% → %v                         ║\n", ascvdResult["riskPercent"], ascvdResult["riskCategory"])
		fmt.Printf("║  CHA₂DS₂-VASc: %.0f → Anticoagulation: %v                       ║\n", cha2ds2Result["total"], cha2ds2Result["anticoagulationRecommended"])
		fmt.Printf("║  HAS-BLED: %.0f → High Bleeding Risk: %v                         ║\n", hasBledResult["total"], hasBledResult["highRisk"])
		fmt.Println("╠═════════════════════════════════════════════════════════════════╣")
		fmt.Println("║  RECOMMENDATIONS:                                               ║")
		fmt.Println("║  • Consider anticoagulation for AFib (stroke risk > bleed risk) ║")
		fmt.Println("║  • High-intensity statin recommended for ASCVD risk             ║")
		fmt.Println("║  • Monitor renal function - dose adjust for CKD                 ║")
		fmt.Println("╚═════════════════════════════════════════════════════════════════╝")
	})
}
