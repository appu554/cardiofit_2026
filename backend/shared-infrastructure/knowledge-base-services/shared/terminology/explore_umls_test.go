// Package terminology - Deep exploration of UMLS API for MedDRA data
//
// This test thoroughly explores what MedDRA data is available in UMLS
// to determine if it meets Phase 3 requirements.
package terminology

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
)

const UMLS_API_KEY = "8ae0c58b-ce41-4d9f-be4d-3ffa77f29480"

// TestUMLS_ExploreMedDRASource explores the MedDRA (MDR) source in UMLS
func TestUMLS_ExploreMedDRASource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	t.Log("╔════════════════════════════════════════════════════════════════╗")
	t.Log("║  DEEP EXPLORATION: UMLS MedDRA (MDR) Source                    ║")
	t.Log("╚════════════════════════════════════════════════════════════════╝")

	// Test 1: Get MedDRA source metadata
	t.Run("GetMedDRASourceInfo", func(t *testing.T) {
		// Query UMLS for MDR source information
		params := url.Values{}
		params.Set("apiKey", UMLS_API_KEY)

		reqURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/content/current/source/MDR?%s", params.Encode())

		resp, err := http.Get(reqURL)
		if err != nil {
			t.Fatalf("Failed to get MDR source info: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("MDR Source Info Response (%d):\n%s", resp.StatusCode, string(body))
	})

	// Test 2: Search for a term and get its MedDRA CODE (not just name)
	t.Run("GetMedDRAPTCode_Nausea", func(t *testing.T) {
		// Search for Nausea in MDR source
		params := url.Values{}
		params.Set("apiKey", UMLS_API_KEY)
		params.Set("string", "Nausea")
		params.Set("sabs", "MDR")
		params.Set("returnIdType", "sourceUi") // Get source-specific ID (MedDRA code)

		reqURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/search/current?%s", params.Encode())

		resp, err := http.Get(reqURL)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		t.Logf("Search for 'Nausea' with returnIdType=sourceUi:\n")
		prettyJSON, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("%s", string(prettyJSON))
	})

	// Test 3: Get atoms for a concept to find MedDRA source codes
	t.Run("GetAtomsWithSourceCodes", func(t *testing.T) {
		// First, search to get a CUI
		params := url.Values{}
		params.Set("apiKey", UMLS_API_KEY)
		params.Set("string", "Nausea")
		params.Set("sabs", "MDR")

		searchURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/search/current?%s", params.Encode())
		resp, err := http.Get(searchURL)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var searchResult map[string]interface{}
		json.Unmarshal(body, &searchResult)

		// Extract CUI from first result
		if resultData, ok := searchResult["result"].(map[string]interface{}); ok {
			if results, ok := resultData["results"].([]interface{}); ok && len(results) > 0 {
				if firstResult, ok := results[0].(map[string]interface{}); ok {
					cui := firstResult["ui"].(string)
					t.Logf("Found CUI: %s for Nausea", cui)

					// Now get atoms for this CUI with MDR source
					atomParams := url.Values{}
					atomParams.Set("apiKey", UMLS_API_KEY)
					atomParams.Set("sabs", "MDR")
					atomParams.Set("language", "ENG")

					atomURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/content/current/CUI/%s/atoms?%s", cui, atomParams.Encode())
					atomResp, err := http.Get(atomURL)
					if err != nil {
						t.Fatalf("Failed to get atoms: %v", err)
					}
					defer atomResp.Body.Close()

					atomBody, _ := io.ReadAll(atomResp.Body)

					var atomResult map[string]interface{}
					json.Unmarshal(atomBody, &atomResult)

					t.Logf("\nAtoms for CUI %s (MDR source):\n", cui)
					prettyJSON, _ := json.MarshalIndent(atomResult, "", "  ")
					t.Logf("%s", string(prettyJSON))
				}
			}
		}
	})

	// Test 4: Query by source code (MedDRA PT code)
	t.Run("QueryByMedDRACode_10028813", func(t *testing.T) {
		// 10028813 is the MedDRA PT code for Nausea
		params := url.Values{}
		params.Set("apiKey", UMLS_API_KEY)

		// Try to get concept by source code
		reqURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/content/current/source/MDR/10028813?%s", params.Encode())

		resp, err := http.Get(reqURL)
		if err != nil {
			t.Fatalf("Failed to query by code: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Query by MedDRA code 10028813 (Nausea):\n%s", string(body))
	})

	// Test 5: Check if UMLS has LLT terms (not just PT)
	t.Run("CheckLLTTerms", func(t *testing.T) {
		// Search for a known LLT that differs from PT
		// "Feeling sick" is an LLT for PT "Nausea"
		params := url.Values{}
		params.Set("apiKey", UMLS_API_KEY)
		params.Set("string", "Feeling sick")
		params.Set("sabs", "MDR")

		reqURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/search/current?%s", params.Encode())

		resp, err := http.Get(reqURL)
		if err != nil {
			t.Fatalf("Failed to search LLT: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Search for LLT 'Feeling sick' (should map to PT Nausea):\n%s", string(body))
	})

	// Test 6: Get term types available in MDR
	t.Run("GetMDRTermTypes", func(t *testing.T) {
		// Search and check TTY (term type) field
		params := url.Values{}
		params.Set("apiKey", UMLS_API_KEY)
		params.Set("string", "Nausea")
		params.Set("sabs", "MDR")

		searchURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/search/current?%s", params.Encode())
		resp, _ := http.Get(searchURL)
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var searchResult map[string]interface{}
		json.Unmarshal(body, &searchResult)

		if resultData, ok := searchResult["result"].(map[string]interface{}); ok {
			if results, ok := resultData["results"].([]interface{}); ok && len(results) > 0 {
				if firstResult, ok := results[0].(map[string]interface{}); ok {
					cui := firstResult["ui"].(string)

					// Get all atoms to see term types
					atomParams := url.Values{}
					atomParams.Set("apiKey", UMLS_API_KEY)
					atomParams.Set("sabs", "MDR")
					atomParams.Set("pageSize", "100")

					atomURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/content/current/CUI/%s/atoms?%s", cui, atomParams.Encode())
					atomResp, _ := http.Get(atomURL)
					defer atomResp.Body.Close()
					atomBody, _ := io.ReadAll(atomResp.Body)

					var atomResult map[string]interface{}
					json.Unmarshal(atomBody, &atomResult)

					t.Log("\nTerm Types (TTY) found in MDR for Nausea:")
					if atomList, ok := atomResult["result"].([]interface{}); ok {
						for _, atom := range atomList {
							if atomMap, ok := atom.(map[string]interface{}); ok {
								tty := atomMap["termType"]
								name := atomMap["name"]
								code := atomMap["sourceConcept"]
								t.Logf("  - TTY: %v, Name: %v, Code: %v", tty, name, code)
							}
						}
					}
				}
			}
		}
	})

	// Test 7: Count total MDR terms available
	t.Run("CountMDRTerms", func(t *testing.T) {
		// Try to get statistics or count
		params := url.Values{}
		params.Set("apiKey", UMLS_API_KEY)
		params.Set("string", "*")
		params.Set("sabs", "MDR")
		params.Set("pageSize", "1")

		reqURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/search/current?%s", params.Encode())

		resp, err := http.Get(reqURL)
		if err != nil {
			t.Logf("Failed to count: %v", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		var result map[string]interface{}
		json.Unmarshal(body, &result)

		if resultData, ok := result["result"].(map[string]interface{}); ok {
			if results, ok := resultData["results"].(map[string]interface{}); ok {
				total := results["total"]
				t.Logf("Total MDR terms in UMLS: %v", total)
			}
		}

		t.Logf("Full response:\n%s", string(body))
	})

	_ = ctx // Use context
}

// TestUMLS_VerifyMedDRACodes verifies we can get actual MedDRA PT codes
func TestUMLS_VerifyMedDRACodes(t *testing.T) {
	t.Log("╔════════════════════════════════════════════════════════════════╗")
	t.Log("║  VERIFY: Can we get MedDRA PT CODES from UMLS?                 ║")
	t.Log("╚════════════════════════════════════════════════════════════════╝")

	testTerms := []struct {
		name         string
		expectedCode string // Known MedDRA PT codes
	}{
		{"Nausea", "10028813"},
		{"Headache", "10019211"},
		{"Diarrhoea", "10012735"},
		{"Arthritis", "10003246"},
		{"Insomnia", "10022437"},
	}

	for _, tt := range testTerms {
		t.Run(fmt.Sprintf("Verify_%s", tt.name), func(t *testing.T) {
			// Search in MDR source
			params := url.Values{}
			params.Set("apiKey", UMLS_API_KEY)
			params.Set("string", tt.name)
			params.Set("sabs", "MDR")
			params.Set("searchType", "exact")

			searchURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/search/current?%s", params.Encode())
			resp, err := http.Get(searchURL)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			var searchResult map[string]interface{}
			json.Unmarshal(body, &searchResult)

			// Extract results
			if resultData, ok := searchResult["result"].(map[string]interface{}); ok {
				if results, ok := resultData["results"].([]interface{}); ok {
					found := false
					for _, r := range results {
						if res, ok := r.(map[string]interface{}); ok {
							cui := res["ui"].(string)
							name := res["name"].(string)

							// The CUI in UMLS for MDR IS the MedDRA code!
							if cui == tt.expectedCode {
								t.Logf("✓ FOUND: %s → MedDRA Code: %s (CUI=%s)", tt.name, tt.expectedCode, cui)
								found = true
								break
							} else {
								t.Logf("  Found: %s (CUI=%s, Name=%s)", tt.name, cui, name)
							}
						}
					}
					if !found {
						t.Logf("⚠ Code %s not directly matched, checking if CUI maps to MedDRA...", tt.expectedCode)
					}
				}
			}

			time.Sleep(200 * time.Millisecond) // Rate limiting
		})
	}
}

// TestUMLS_GetSourceConceptDirectly tries to get MedDRA source concept directly
func TestUMLS_GetSourceConceptDirectly(t *testing.T) {
	t.Log("╔════════════════════════════════════════════════════════════════╗")
	t.Log("║  DIRECT ACCESS: Query UMLS by MedDRA Source Code               ║")
	t.Log("╚════════════════════════════════════════════════════════════════╝")

	codes := []struct {
		code string
		name string
	}{
		{"10028813", "Nausea"},
		{"10019211", "Headache"},
		{"10012735", "Diarrhoea"},
	}

	for _, c := range codes {
		t.Run(fmt.Sprintf("DirectLookup_%s", c.code), func(t *testing.T) {
			params := url.Values{}
			params.Set("apiKey", UMLS_API_KEY)

			// Direct source concept lookup
			reqURL := fmt.Sprintf("https://uts-ws.nlm.nih.gov/rest/content/current/source/MDR/%s?%s", c.code, params.Encode())

			resp, err := http.Get(reqURL)
			if err != nil {
				t.Fatalf("Lookup failed: %v", err)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode == 200 {
				var result map[string]interface{}
				json.Unmarshal(body, &result)

				if resultData, ok := result["result"].(map[string]interface{}); ok {
					name := resultData["name"]
					ui := resultData["ui"]
					rootSource := resultData["rootSource"]
					t.Logf("✓ Code %s → Name: %v, UI: %v, Source: %v", c.code, name, ui, rootSource)
				} else {
					t.Logf("Response: %s", string(body))
				}
			} else {
				t.Logf("✗ Code %s not found (status %d): %s", c.code, resp.StatusCode, string(body))
			}

			time.Sleep(200 * time.Millisecond)
		})
	}
}
