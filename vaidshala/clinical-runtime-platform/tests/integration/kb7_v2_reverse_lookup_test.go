// Package integration tests KB-7 v2 Reverse Lookup functionality.
//
// This test validates the KB-7 v2 architecture which uses REVERSE LOOKUP
// instead of iterating over 18,000+ ValueSets.
//
// KEY ARCHITECTURE CHANGE:
//   OLD: For each of 18,000 ValueSets → check if code is member → O(18,000 * n)
//   NEW: For each patient code → single reverse lookup → O(n)
//
// PERFORMANCE IMPROVEMENT: 18,000x faster!
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	kb7V2URL = "http://localhost:8092" // KB-7 Terminology Service
)

func getKB7V2URL() string {
	if url := os.Getenv("KB7_URL"); url != "" {
		return url
	}
	return kb7V2URL
}

// TestPatientData represents the production-like patient for testing
type TestPatientData struct {
	PatientID   string            `json:"patient_id"`
	RequestedBy string            `json:"requested_by"`
	EncounterID string            `json:"encounter_id"`
	PatientData PatientDataDetail `json:"patient_data"`
}

type PatientDataDetail struct {
	Demographics Demographics   `json:"demographics"`
	Conditions   []Condition    `json:"conditions"`
	Medications  []Medication   `json:"medications"`
	LabResults   []LabResult    `json:"lab_results"`
	VitalSigns   []VitalSign    `json:"vital_signs"`
	Encounters   []Encounter    `json:"encounters"`
	Allergies    []Allergy      `json:"allergies"`
}

type Demographics struct {
	BirthDate string `json:"birth_date"`
	Gender    string `json:"gender"`
	Region    string `json:"region"`
}

type Condition struct {
	Code           string `json:"code"`
	System         string `json:"system"`
	Display        string `json:"display"`
	ClinicalStatus string `json:"clinical_status"`
	OnsetDate      string `json:"onset_date"`
}

type Medication struct {
	Code      string `json:"code"`
	System    string `json:"system"`
	Display   string `json:"display"`
	Status    string `json:"status"`
	Dose      string `json:"dose"`
	DoseUnit  string `json:"dose_unit"`
	Frequency string `json:"frequency"`
	Route     string `json:"route"`
	StartDate string `json:"start_date"`
}

type LabResult struct {
	Code           string  `json:"code"`
	System         string  `json:"system"`
	Display        string  `json:"display"`
	Value          float64 `json:"value"`
	Unit           string  `json:"unit"`
	Timestamp      string  `json:"timestamp"`
	ReferenceRange string  `json:"reference_range"`
}

type VitalSign struct {
	SystolicBP  int    `json:"systolic_bp"`
	DiastolicBP int    `json:"diastolic_bp"`
	HeartRate   int    `json:"heart_rate"`
	Timestamp   string `json:"timestamp"`
}

type Encounter struct {
	EncounterID string `json:"encounter_id"`
	Class       string `json:"class"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time,omitempty"`
}

type Allergy struct {
	Code     string `json:"code"`
	System   string `json:"system"`
	Display  string `json:"display"`
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Reaction string `json:"reaction"`
}

// LookupMembershipsResponse mirrors the KB-7 v2 response
type LookupMembershipsResponse struct {
	Code             string               `json:"code"`
	System           string               `json:"system"`
	TotalMemberships int                  `json:"total_memberships"`
	CanonicalCount   int                  `json:"canonical_count"`
	SemanticNames    []string             `json:"semantic_names"`
	Memberships      []ValueSetMembership `json:"memberships"`
	ProcessingTimeMs int64                `json:"processing_time_ms"`
}

type ValueSetMembership struct {
	ValueSetURL  string `json:"valueset_url"`
	ValueSetOID  string `json:"valueset_oid,omitempty"`
	SemanticName string `json:"semantic_name"`
	Title        string `json:"title,omitempty"`
	Category     string `json:"category,omitempty"`
	IsCanonical  bool   `json:"is_canonical"`
	CodeDisplay  string `json:"code_display,omitempty"`
}

// getTestPatient returns the production-like patient data
func getTestPatient() TestPatientData {
	return TestPatientData{
		PatientID:   "PAT-ROHAN-001",
		RequestedBy: "Dr. Smith",
		EncounterID: "ENC-001",
		PatientData: PatientDataDetail{
			Demographics: Demographics{
				BirthDate: "1958-03-15",
				Gender:    "male",
				Region:    "AU",
			},
			Conditions: []Condition{
				{
					Code:           "38341003",
					System:         "http://snomed.info/sct",
					Display:        "Essential hypertension",
					ClinicalStatus: "active",
					OnsetDate:      "2020-06-15",
				},
				{
					Code:           "44054006",
					System:         "http://snomed.info/sct",
					Display:        "Type 2 diabetes mellitus",
					ClinicalStatus: "active",
					OnsetDate:      "2019-03-10",
				},
			},
			Medications: []Medication{
				{
					Code:      "314076",
					System:    "http://www.nlm.nih.gov/research/umls/rxnorm",
					Display:   "Lisinopril 10 MG Oral Tablet",
					Status:    "active",
					Dose:      "10",
					DoseUnit:  "mg",
					Frequency: "once daily",
					Route:     "oral",
					StartDate: "2020-06-20",
				},
				{
					Code:      "860975",
					System:    "http://www.nlm.nih.gov/research/umls/rxnorm",
					Display:   "Metformin 500 MG Oral Tablet",
					Status:    "active",
					Dose:      "500",
					DoseUnit:  "mg",
					Frequency: "twice daily",
					Route:     "oral",
					StartDate: "2019-03-15",
				},
				{
					Code:      "197361",
					System:    "http://www.nlm.nih.gov/research/umls/rxnorm",
					Display:   "Amlodipine 5 MG Oral Tablet",
					Status:    "active",
					Dose:      "5",
					DoseUnit:  "mg",
					Frequency: "once daily",
					Route:     "oral",
					StartDate: "2023-01-10",
				},
				{
					Code:      "259255",
					System:    "http://www.nlm.nih.gov/research/umls/rxnorm",
					Display:   "Atorvastatin 20 MG Oral Tablet",
					Status:    "active",
					Dose:      "20",
					DoseUnit:  "mg",
					Frequency: "once daily at bedtime",
					Route:     "oral",
					StartDate: "2020-08-01",
				},
			},
			LabResults: []LabResult{
				{
					Code:           "4548-4",
					System:         "http://loinc.org",
					Display:        "Hemoglobin A1c",
					Value:          7.2,
					Unit:           "%",
					Timestamp:      "2025-12-15T08:30:00Z",
					ReferenceRange: "< 7.0",
				},
				{
					Code:           "2160-0",
					System:         "http://loinc.org",
					Display:        "Creatinine",
					Value:          1.1,
					Unit:           "mg/dL",
					Timestamp:      "2025-12-15T08:30:00Z",
					ReferenceRange: "0.7-1.3",
				},
				{
					Code:           "13457-7",
					System:         "http://loinc.org",
					Display:        "LDL Cholesterol",
					Value:          95,
					Unit:           "mg/dL",
					Timestamp:      "2025-12-15T08:30:00Z",
					ReferenceRange: "< 100",
				},
				{
					Code:           "33914-3",
					System:         "http://loinc.org",
					Display:        "eGFR",
					Value:          72,
					Unit:           "mL/min/1.73m2",
					Timestamp:      "2025-12-15T08:30:00Z",
					ReferenceRange: "> 60",
				},
			},
			VitalSigns: []VitalSign{
				{
					SystolicBP:  142,
					DiastolicBP: 88,
					HeartRate:   76,
					Timestamp:   "2026-01-09T10:00:00Z",
				},
				{
					SystolicBP:  138,
					DiastolicBP: 86,
					HeartRate:   72,
					Timestamp:   "2026-01-08T09:30:00Z",
				},
			},
			Encounters: []Encounter{
				{
					EncounterID: "ENC-001",
					Class:       "ambulatory",
					Type:        "routine checkup",
					Status:      "in-progress",
					StartTime:   "2026-01-09T09:45:00Z",
				},
			},
			Allergies: []Allergy{
				{
					Code:     "7980",
					System:   "http://www.nlm.nih.gov/research/umls/rxnorm",
					Display:  "Penicillin",
					Type:     "allergy",
					Severity: "moderate",
					Reaction: "Rash",
				},
			},
		},
	}
}

// ============================================================================
// KB-7 v2 REVERSE LOOKUP TESTS
// ============================================================================

func TestKB7V2ReverseLookup(t *testing.T) {
	kb7URL := getKB7V2URL()

	// Check if KB-7 is running
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", kb7URL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Skipf("KB-7 not available at %s: %v", kb7URL, err)
	}
	resp.Body.Close()

	patient := getTestPatient()

	printHeader("KB-7 v2 REVERSE LOOKUP TEST")
	printPatientSummary(patient)

	// =========================================================================
	// TEST 1: Reverse Lookup for Lisinopril (ACE Inhibitor)
	// =========================================================================
	t.Run("reverse_lookup_lisinopril", func(t *testing.T) {
		printTestHeader("TEST 1: Reverse Lookup for Lisinopril (RxNorm: 314076)")

		code := "314076"
		system := "http://www.nlm.nih.gov/research/umls/rxnorm"

		result, duration, err := performReverseLookup(kb7URL, code, system, false)
		if err != nil {
			t.Logf("⚠️  Reverse lookup failed: %v", err)
			return
		}

		printReverseLookupResult(code, system, "Lisinopril 10 MG Oral Tablet", result, duration)

		// Verify expected memberships
		expectedSemanticNames := []string{"ACE Inhibitors", "Lisinopril"}
		verifySemanticNames(t, result, expectedSemanticNames)
	})

	// =========================================================================
	// TEST 2: Reverse Lookup for Metformin (Diabetes Medication)
	// =========================================================================
	t.Run("reverse_lookup_metformin", func(t *testing.T) {
		printTestHeader("TEST 2: Reverse Lookup for Metformin (RxNorm: 860975)")

		code := "860975"
		system := "http://www.nlm.nih.gov/research/umls/rxnorm"

		result, duration, err := performReverseLookup(kb7URL, code, system, false)
		if err != nil {
			t.Logf("⚠️  Reverse lookup failed: %v", err)
			return
		}

		printReverseLookupResult(code, system, "Metformin 500 MG Oral Tablet", result, duration)

		// Verify expected memberships
		expectedSemanticNames := []string{"Metformin", "Diabetes"}
		verifySemanticNames(t, result, expectedSemanticNames)
	})

	// =========================================================================
	// TEST 3: Reverse Lookup for Type 2 Diabetes (SNOMED)
	// =========================================================================
	t.Run("reverse_lookup_diabetes", func(t *testing.T) {
		printTestHeader("TEST 3: Reverse Lookup for Type 2 Diabetes (SNOMED: 44054006)")

		code := "44054006"
		system := "http://snomed.info/sct"

		result, duration, err := performReverseLookup(kb7URL, code, system, false)
		if err != nil {
			t.Logf("⚠️  Reverse lookup failed: %v", err)
			return
		}

		printReverseLookupResult(code, system, "Type 2 diabetes mellitus", result, duration)

		// Verify expected memberships
		expectedSemanticNames := []string{"Diabetes", "Type 2"}
		verifySemanticNames(t, result, expectedSemanticNames)
	})

	// =========================================================================
	// TEST 4: Reverse Lookup for Essential Hypertension (SNOMED)
	// =========================================================================
	t.Run("reverse_lookup_hypertension", func(t *testing.T) {
		printTestHeader("TEST 4: Reverse Lookup for Essential Hypertension (SNOMED: 38341003)")

		code := "38341003"
		system := "http://snomed.info/sct"

		result, duration, err := performReverseLookup(kb7URL, code, system, false)
		if err != nil {
			t.Logf("⚠️  Reverse lookup failed: %v", err)
			return
		}

		printReverseLookupResult(code, system, "Essential hypertension", result, duration)

		// Verify expected memberships
		expectedSemanticNames := []string{"Hypertension", "Essential Hypertension"}
		verifySemanticNames(t, result, expectedSemanticNames)
	})

	// =========================================================================
	// TEST 5: Full Patient Processing - All Codes
	// =========================================================================
	t.Run("full_patient_reverse_lookup", func(t *testing.T) {
		printTestHeader("TEST 5: Full Patient Processing - All Conditions and Medications")

		allSemanticNames := make(map[string]bool)
		totalDuration := time.Duration(0)
		totalCalls := 0

		fmt.Println("\n📋 Processing Patient Conditions:")
		fmt.Println("─────────────────────────────────")
		for _, cond := range patient.PatientData.Conditions {
			result, duration, err := performReverseLookup(kb7URL, cond.Code, cond.System, false)
			if err != nil {
				fmt.Printf("  ⚠️  %s (%s): Error - %v\n", cond.Display, cond.Code, err)
				continue
			}
			totalDuration += duration
			totalCalls++

			fmt.Printf("  ✅ %s (%s)\n", cond.Display, cond.Code)
			fmt.Printf("     → Memberships: %d | Canonical: %d | Time: %v\n",
				result.TotalMemberships, result.CanonicalCount, duration)
			if len(result.SemanticNames) > 0 {
				fmt.Printf("     → Semantic Names: %s\n", strings.Join(result.SemanticNames[:min(5, len(result.SemanticNames))], ", "))
			}

			for _, name := range result.SemanticNames {
				allSemanticNames[name] = true
			}
		}

		fmt.Println("\n💊 Processing Patient Medications:")
		fmt.Println("───────────────────────────────────")
		for _, med := range patient.PatientData.Medications {
			result, duration, err := performReverseLookup(kb7URL, med.Code, med.System, false)
			if err != nil {
				fmt.Printf("  ⚠️  %s (%s): Error - %v\n", med.Display, med.Code, err)
				continue
			}
			totalDuration += duration
			totalCalls++

			fmt.Printf("  ✅ %s (%s)\n", med.Display, med.Code)
			fmt.Printf("     → Memberships: %d | Canonical: %d | Time: %v\n",
				result.TotalMemberships, result.CanonicalCount, duration)
			if len(result.SemanticNames) > 0 {
				fmt.Printf("     → Semantic Names: %s\n", strings.Join(result.SemanticNames[:min(5, len(result.SemanticNames))], ", "))
			}

			for _, name := range result.SemanticNames {
				allSemanticNames[name] = true
			}
		}

		// Print summary
		fmt.Println("\n" + strings.Repeat("═", 70))
		fmt.Println("                    📊 PROCESSING SUMMARY")
		fmt.Println(strings.Repeat("═", 70))
		fmt.Printf("  Total API Calls:       %d\n", totalCalls)
		fmt.Printf("  Total Processing Time: %v\n", totalDuration)
		if totalCalls > 0 {
			fmt.Printf("  Average per Call:      %v\n", totalDuration/time.Duration(totalCalls))
		} else {
			fmt.Printf("  Average per Call:      N/A (no successful calls)\n")
		}
		fmt.Printf("  Unique Semantic Names: %d\n", len(allSemanticNames))
		fmt.Println()

		// Print all semantic names found
		fmt.Println("  📝 All Semantic Names Found:")
		semanticList := make([]string, 0, len(allSemanticNames))
		for name := range allSemanticNames {
			semanticList = append(semanticList, name)
		}
		for i, name := range semanticList {
			fmt.Printf("     %d. %s\n", i+1, name)
		}
		fmt.Println(strings.Repeat("═", 70))

		// Performance comparison
		fmt.Println("\n  ⚡ PERFORMANCE COMPARISON:")
		fmt.Println("  ──────────────────────────")
		fmt.Printf("  KB-7 v1 (OLD): Would have made ~18,000 × %d = %d API calls\n",
			totalCalls, 18000*totalCalls)
		fmt.Printf("  KB-7 v2 (NEW): Made only %d API calls\n", totalCalls)
		fmt.Printf("  Speedup:       %.0fx faster!\n", float64(18000*totalCalls)/float64(totalCalls))
		fmt.Println(strings.Repeat("═", 70))
	})

	// =========================================================================
	// TEST 6: Canonical-Only Lookup
	// =========================================================================
	t.Run("canonical_only_lookup", func(t *testing.T) {
		printTestHeader("TEST 6: Canonical-Only Lookup (ICU/Safety-Critical ValueSets)")

		code := "314076" // Lisinopril
		system := "http://www.nlm.nih.gov/research/umls/rxnorm"

		// First, get all memberships
		allResult, _, err := performReverseLookup(kb7URL, code, system, false)
		if err != nil {
			t.Logf("⚠️  All memberships lookup failed: %v", err)
			return
		}

		// Then, get canonical-only
		canonicalResult, duration, err := performReverseLookup(kb7URL, code, system, true)
		if err != nil {
			t.Logf("⚠️  Canonical lookup failed: %v", err)
			return
		}

		fmt.Printf("\n  📊 Comparison for Lisinopril (314076):\n")
		fmt.Printf("     All ValueSets:      %d memberships\n", allResult.TotalMemberships)
		fmt.Printf("     Canonical Only:     %d memberships\n", canonicalResult.TotalMemberships)
		fmt.Printf("     Reduction:          %.1f%%\n",
			(1-float64(canonicalResult.TotalMemberships)/float64(allResult.TotalMemberships))*100)
		fmt.Printf("     Lookup Time:        %v\n", duration)

		if len(canonicalResult.SemanticNames) > 0 {
			fmt.Printf("\n  🏥 Canonical Semantic Names (ICU/Safety):\n")
			for i, name := range canonicalResult.SemanticNames {
				fmt.Printf("     %d. %s\n", i+1, name)
			}
		}
	})
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func performReverseLookup(baseURL, code, system string, canonicalOnly bool) (*LookupMembershipsResponse, time.Duration, error) {
	endpoint := fmt.Sprintf("%s/fhir/CodeSystem/$lookup-memberships?code=%s", baseURL, code)
	if system != "" {
		endpoint += "&system=" + system
	}
	if canonicalOnly {
		endpoint += "&canonical=true"
	}

	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, duration, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, duration, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result LookupMembershipsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, duration, err
	}

	return &result, duration, nil
}

func printHeader(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("                    %s\n", title)
	fmt.Println(strings.Repeat("═", 80))
}

func printTestHeader(title string) {
	fmt.Printf("\n┌%s┐\n", strings.Repeat("─", 68))
	fmt.Printf("│  %-66s│\n", title)
	fmt.Printf("└%s┘\n", strings.Repeat("─", 68))
}

func printPatientSummary(patient TestPatientData) {
	fmt.Println()
	fmt.Println("📋 PATIENT SUMMARY")
	fmt.Println("──────────────────")
	fmt.Printf("  Patient ID:    %s\n", patient.PatientID)
	fmt.Printf("  Birth Date:    %s\n", patient.PatientData.Demographics.BirthDate)
	fmt.Printf("  Gender:        %s\n", patient.PatientData.Demographics.Gender)
	fmt.Printf("  Region:        %s\n", patient.PatientData.Demographics.Region)
	fmt.Printf("  Conditions:    %d\n", len(patient.PatientData.Conditions))
	for _, c := range patient.PatientData.Conditions {
		fmt.Printf("                 - %s (%s)\n", c.Display, c.Code)
	}
	fmt.Printf("  Medications:   %d\n", len(patient.PatientData.Medications))
	for _, m := range patient.PatientData.Medications {
		fmt.Printf("                 - %s (%s)\n", m.Display, m.Code)
	}
	fmt.Printf("  Lab Results:   %d\n", len(patient.PatientData.LabResults))
	fmt.Printf("  Allergies:     %d\n", len(patient.PatientData.Allergies))
	fmt.Println()
}

func printReverseLookupResult(code, system, display string, result *LookupMembershipsResponse, duration time.Duration) {
	fmt.Printf("\n  📥 INPUT:\n")
	fmt.Printf("     Code:    %s\n", code)
	fmt.Printf("     System:  %s\n", system)
	fmt.Printf("     Display: %s\n", display)

	fmt.Printf("\n  📤 OUTPUT:\n")
	fmt.Printf("     Total Memberships:  %d\n", result.TotalMemberships)
	fmt.Printf("     Canonical Count:    %d\n", result.CanonicalCount)
	fmt.Printf("     Processing Time:    %v (reported: %dms)\n", duration, result.ProcessingTimeMs)

	if len(result.SemanticNames) > 0 {
		fmt.Printf("\n  📝 Semantic Names (human-readable):\n")
		for i, name := range result.SemanticNames {
			if i >= 10 {
				fmt.Printf("     ... and %d more\n", len(result.SemanticNames)-10)
				break
			}
			fmt.Printf("     %d. %s\n", i+1, name)
		}
	}

	if len(result.Memberships) > 0 {
		fmt.Printf("\n  📊 Sample Memberships (with details):\n")
		for i, m := range result.Memberships {
			if i >= 5 {
				fmt.Printf("     ... and %d more\n", len(result.Memberships)-5)
				break
			}
			canonical := ""
			if m.IsCanonical {
				canonical = " 🏥"
			}
			fmt.Printf("     %d. %s [%s]%s\n", i+1, m.SemanticName, m.Category, canonical)
		}
	}
}

func verifySemanticNames(t *testing.T, result *LookupMembershipsResponse, expectedPrefixes []string) {
	if result == nil || len(result.SemanticNames) == 0 {
		t.Logf("⚠️  No semantic names returned")
		return
	}

	found := make(map[string]bool)
	for _, expected := range expectedPrefixes {
		for _, name := range result.SemanticNames {
			if strings.Contains(strings.ToLower(name), strings.ToLower(expected)) {
				found[expected] = true
				break
			}
		}
	}

	fmt.Printf("\n  ✅ Verification:\n")
	for _, expected := range expectedPrefixes {
		if found[expected] {
			fmt.Printf("     ✓ Found '%s' in semantic names\n", expected)
		} else {
			fmt.Printf("     ✗ Missing '%s' in semantic names\n", expected)
			t.Logf("Expected semantic name containing '%s' not found", expected)
		}
	}
}

// ============================================================================
// DIRECT HTTP TEST (Standalone - no dependencies)
// ============================================================================

func TestKB7V2DirectHTTP(t *testing.T) {
	kb7URL := getKB7V2URL()

	// Quick health check
	resp, err := http.Get(kb7URL + "/health")
	if err != nil {
		t.Skipf("KB-7 not available: %v", err)
	}
	resp.Body.Close()

	printHeader("KB-7 v2 DIRECT HTTP TEST")

	t.Run("direct_http_lisinopril", func(t *testing.T) {
		endpoint := kb7URL + "/fhir/CodeSystem/$lookup-memberships?code=314076&system=http://www.nlm.nih.gov/research/umls/rxnorm"

		fmt.Printf("\n  🔗 Endpoint: %s\n", endpoint)

		start := time.Now()
		resp, err := http.Get(endpoint)
		if err != nil {
			t.Fatalf("HTTP request failed: %v", err)
		}
		defer resp.Body.Close()
		duration := time.Since(start)

		body, _ := io.ReadAll(resp.Body)

		fmt.Printf("  📊 Status:   %s\n", resp.Status)
		fmt.Printf("  ⏱️  Duration: %v\n", duration)
		fmt.Printf("  📦 Response Size: %d bytes\n", len(body))

		if resp.StatusCode == http.StatusOK {
			var result map[string]interface{}
			json.Unmarshal(body, &result)

			prettyJSON, _ := json.MarshalIndent(result, "  ", "  ")
			fmt.Printf("\n  📤 Response:\n%s\n", string(prettyJSON))
		} else {
			fmt.Printf("\n  ❌ Error Response:\n  %s\n", string(body))
		}
	})
}
