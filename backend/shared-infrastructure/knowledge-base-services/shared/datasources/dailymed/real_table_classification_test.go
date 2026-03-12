// Package dailymed - Real SPL Table Classification Test
// Tests table classification on actual FDA SPL documents from DailyMed
package dailymed

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestRealSPLTableClassification tests table classification on real FDA SPL data
func TestRealSPLTableClassification(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create fetcher
	config := DefaultConfig()
	config.Timeout = 60 * time.Second
	cache := NewMemorySPLCache(1 * time.Hour)
	fetcher := NewSPLFetcher(config, cache)

	// Create classifier
	classifier := NewTableClassifier()

	// Search for metformin to get a valid SetID
	t.Log("🔍 Searching for metformin SPL...")
	results, err := fetcher.SearchByDrugName(ctx, "metformin")
	if err != nil {
		t.Fatalf("Failed to search for metformin: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("No SPL results found for metformin")
	}

	setID := results[0].SetID
	t.Logf("✅ Found SetID: %s", setID)
	t.Logf("   Title: %s", results[0].Title)

	// Fetch the full SPL document
	t.Log("\n📥 Fetching full SPL document...")
	doc, err := fetcher.FetchBySetID(ctx, setID)
	if err != nil {
		t.Fatalf("Failed to fetch SPL: %v", err)
	}

	t.Logf("✅ Fetched SPL: %s", doc.Title)
	t.Logf("   Version: %d", doc.VersionNumber.Value)
	t.Logf("   Sections: %d top-level sections", len(doc.Sections))

	// Get all sections with LOINC codes
	allSections := doc.GetAllSections()
	t.Logf("   Total sections (including subsections): %d", len(allSections))

	// Find all sections with tables
	t.Log("\n" + strings.Repeat("=", 80))
	t.Log("📊 ANALYZING ALL TABLES IN SPL DOCUMENT")
	t.Log(strings.Repeat("=", 80))

	tableCount := 0
	sectionsWithTables := 0

	for loincCode, section := range allSections {
		if !section.HasTables() {
			continue
		}

		sectionsWithTables++
		tables := section.GetTables()

		t.Logf("\n🏷️  SECTION: %s", section.Title)
		t.Logf("   LOINC Code: %s", loincCode)
		t.Logf("   Table Count: %d", len(tables))

		// Classify each table in this section
		for i, table := range tables {
			tableCount++
			t.Logf("\n   ─── Table %d ───", i+1)

			// Get headers using the method
			headers := table.GetHeaders()

			// Fallback: check for td cells in thead
			if len(headers) == 0 && table.THead.Row.Cells != nil {
				for _, cell := range table.THead.Row.Cells {
					headers = append(headers, stripXMLTags(cell.Content))
				}
			}

			// Another fallback: first row might be headers
			if len(headers) == 0 && len(table.Rows) > 0 {
				t.Log("      ⚠️  No thead headers found, checking first row as potential headers")
				for _, cell := range table.Rows[0].Cells {
					content := stripXMLTags(cell.Content)
					if content != "" {
						headers = append(headers, content)
					}
				}
				for _, cell := range table.Rows[0].HeaderCells {
					content := stripXMLTags(cell.Content)
					if content != "" {
						headers = append(headers, content)
					}
				}
			}

			t.Logf("      ID: %s", table.ID)
			t.Logf("      Caption: %s", table.Caption)
			t.Logf("      Headers (%d): %v", len(headers), truncateStrings(headers, 50))
			t.Logf("      Rows: %d", len(table.Rows))

			// Show first few rows content for context
			if len(table.Rows) > 0 {
				t.Log("      Sample content (first 3 rows):")
				for rowIdx := 0; rowIdx < minInt(3, len(table.Rows)); rowIdx++ {
					row := table.Rows[rowIdx]
					var cells []string
					for _, cell := range row.Cells {
						content := stripXMLTags(cell.Content)
						if len(content) > 40 {
							content = content[:40] + "..."
						}
						cells = append(cells, content)
					}
					// Also check HeaderCells in rows (some tables have th in tbody)
					for _, cell := range row.HeaderCells {
						content := stripXMLTags(cell.Content)
						if len(content) > 40 {
							content = content[:40] + "..."
						}
						cells = append(cells, "[TH:"+content+"]")
					}
					t.Logf("        Row %d: %v", rowIdx+1, cells)
				}
			}

			// Run classification
			result := classifier.ClassifyTable(&table)

			t.Log("\n      📋 CLASSIFICATION RESULT:")
			t.Logf("         Type: %s", result.TableType)
			t.Logf("         Confidence: %.2f (%.0f%%)", result.Confidence, result.Confidence*100)
			if result.AlternativeType != "" {
				t.Logf("         Alternative: %s", result.AlternativeType)
			}

			// Show KB routing
			targetKBs := GetTargetKBsForTableType(result.TableType)
			priority := GetExtractionPriority(result.TableType)
			t.Logf("         Target KBs: %v", targetKBs)
			t.Logf("         Priority: %s", priority)
		}
	}

	t.Log("\n" + strings.Repeat("=", 80))
	t.Log("📊 SUMMARY")
	t.Log(strings.Repeat("=", 80))
	t.Logf("   Total sections: %d", len(allSections))
	t.Logf("   Sections with tables: %d", sectionsWithTables)
	t.Logf("   Total tables found: %d", tableCount)
}

// TestRealDDITableClassification specifically tests Drug Interaction tables
func TestRealDDITableClassification(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create fetcher and classifier
	config := DefaultConfig()
	config.Timeout = 60 * time.Second
	cache := NewMemorySPLCache(1 * time.Hour)
	fetcher := NewSPLFetcher(config, cache)
	classifier := NewTableClassifier()

	// Search for a drug known to have DDI tables
	// Try ritonavir (HIV protease inhibitor with many interactions)
	drugsToTest := []string{"ritonavir", "warfarin", "metformin", "amiodarone"}

	for _, drugName := range drugsToTest {
		t.Logf("\n🔍 Testing %s for DDI tables...", strings.ToUpper(drugName))

		results, err := fetcher.SearchByDrugName(ctx, drugName)
		if err != nil || len(results) == 0 {
			t.Logf("   ⚠️  Could not find SPL for %s", drugName)
			continue
		}

		doc, err := fetcher.FetchBySetID(ctx, results[0].SetID)
		if err != nil {
			t.Logf("   ⚠️  Could not fetch SPL: %v", err)
			continue
		}

		// Get Drug Interactions section
		ddiSection := doc.GetDrugInteractions()
		if ddiSection == nil {
			t.Logf("   ℹ️  No Drug Interactions section found")
			continue
		}

		t.Logf("   ✅ Found Drug Interactions section")

		if !ddiSection.HasTables() {
			t.Logf("   ℹ️  No tables in Drug Interactions section")

			// Show text content snippet
			rawText := ddiSection.GetRawText()
			if len(rawText) > 200 {
				rawText = rawText[:200] + "..."
			}
			t.Logf("   Text content: %s", rawText)
			continue
		}

		tables := ddiSection.GetTables()
		t.Logf("   📊 Found %d tables in DDI section", len(tables))

		for i, table := range tables {
			result := classifier.ClassifyTable(&table)

			t.Logf("\n   Table %d:", i+1)
			t.Logf("      Classification: %s (confidence: %.2f)", result.TableType, result.Confidence)
			t.Logf("      Headers: %v", truncateStrings(table.GetHeaders(), 30))
			t.Logf("      Rows: %d", len(table.Rows))

			// Verify DDI classification
			if result.TableType == TableTypeDDI {
				t.Logf("      ✅ Correctly classified as DDI table!")
			} else if result.Confidence < 0.3 {
				t.Logf("      ⚠️  Low confidence - may need manual review")
			}
		}
	}
}

// TestExtractAndClassifyRealTables tests the full extraction pipeline
func TestExtractAndClassifyRealTables(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	config := DefaultConfig()
	config.Timeout = 60 * time.Second
	cache := NewMemorySPLCache(1 * time.Hour)
	fetcher := NewSPLFetcher(config, cache)
	classifier := NewTableClassifier()

	// Search for metformin
	results, err := fetcher.SearchByDrugName(ctx, "metformin")
	if err != nil || len(results) == 0 {
		t.Skip("Could not find metformin SPL")
	}

	doc, err := fetcher.FetchBySetID(ctx, results[0].SetID)
	if err != nil {
		t.Fatalf("Failed to fetch SPL: %v", err)
	}

	t.Logf("📥 Fetched: %s", doc.Title)

	// Use ExtractAndClassifyTables on each section
	allSections := doc.GetAllSections()

	t.Log("\n📊 EXTRACTION PIPELINE RESULTS:")
	t.Log(strings.Repeat("─", 60))

	var allExtracted []*ExtractedTable

	for loincCode, section := range allSections {
		if !section.HasTables() {
			continue
		}

		extracted := classifier.ExtractAndClassifyTables(section)
		allExtracted = append(allExtracted, extracted...)

		t.Logf("\nSection: %s (LOINC: %s)", section.Title, loincCode)
		for _, et := range extracted {
			t.Logf("  Table ID: %s", et.TableID)
			t.Logf("    Type: %s (%.0f%% confidence)", et.TableType, et.Confidence*100)
			t.Logf("    Headers: %v", truncateStrings(et.Headers, 30))
			t.Logf("    Rows: %d", len(et.Rows))
			t.Logf("    Target KBs: %v", et.TargetKBs)
			t.Logf("    Priority: %s", et.Priority)
		}
	}

	t.Log("\n" + strings.Repeat("─", 60))
	t.Logf("Total extracted tables: %d", len(allExtracted))

	// Summary by type
	typeCounts := make(map[TableType]int)
	for _, et := range allExtracted {
		typeCounts[et.TableType]++
	}

	t.Log("\nBy Classification Type:")
	for tableType, count := range typeCounts {
		t.Logf("  %s: %d tables", tableType, count)
	}
}

// Helper functions

func truncateStrings(strs []string, maxLen int) []string {
	result := make([]string, len(strs))
	for i, s := range strs {
		if len(s) > maxLen {
			result[i] = s[:maxLen] + "..."
		} else {
			result[i] = s
		}
	}
	return result
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
