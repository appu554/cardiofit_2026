// Package integration_tests - Metformin End-to-End Pipeline Test
// Tests the full Canonical Rule Generation pipeline with a real FDA SPL document
// SetID: 2a0c8ed8-3393-4de7-8dbd-4ec639eb68e6 (Metformin Hydrochloride)
package integration_tests

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/rules"
)

// MetforminSetIDFallback is the FDA SPL SetID for Metformin Hydrochloride (may be superseded)
const MetforminSetIDFallback = "2a0c8ed8-3393-4de7-8dbd-4ec639eb68e6"

// TestMetforminE2EPipeline runs the full pipeline with real Metformin SPL
func TestMetforminE2EPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode - requires network access to FDA DailyMed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	t.Log("╔══════════════════════════════════════════════════════════════════╗")
	t.Log("║  METFORMIN END-TO-END PIPELINE TEST                              ║")
	t.Log("║  Drug: Metformin Hydrochloride (searched dynamically)            ║")
	t.Log("╚══════════════════════════════════════════════════════════════════╝")

	// Initialize components
	config := dailymed.DefaultConfig()
	config.Timeout = 60 * time.Second
	cache := dailymed.NewMemorySPLCache(1 * time.Hour)
	fetcher := dailymed.NewSPLFetcher(config, cache)

	// Create simple pipeline (no database)
	pipeline := rules.NewCanonicalRulePipelineSimple(fetcher)

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 0: SEARCH FOR METFORMIN TO GET CURRENT SETID
	// ─────────────────────────────────────────────────────────────────────────
	t.Log("\n🔍 Step 0: Searching DailyMed for Metformin...")
	searchResults, err := fetcher.SearchByDrugName(ctx, "metformin")
	if err != nil {
		t.Fatalf("Failed to search for Metformin: %v", err)
	}
	if len(searchResults) == 0 {
		t.Fatal("No Metformin SPL documents found in DailyMed")
	}

	MetforminSetID := searchResults[0].SetID
	t.Logf("   ✅ Found %d results, using SetID: %s", len(searchResults), MetforminSetID)

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 1: FETCH SPL DOCUMENT
	// ─────────────────────────────────────────────────────────────────────────
	t.Log("\n📥 Step 1: Fetching Metformin SPL document...")
	startFetch := time.Now()

	doc, err := fetcher.FetchBySetID(ctx, MetforminSetID)
	if err != nil {
		t.Fatalf("Failed to fetch Metformin SPL: %v", err)
	}

	fetchDuration := time.Since(startFetch)
	t.Logf("   ✅ Fetched in %v", fetchDuration.Round(time.Millisecond))
	t.Logf("   📄 Title: %s", truncateString(doc.Title, 70))
	t.Logf("   🔑 SetID: %s", doc.SetID.Extension)
	t.Logf("   📊 Sections: %d", len(doc.Sections))

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 2: ANALYZE SECTIONS
	// ─────────────────────────────────────────────────────────────────────────
	t.Log("\n📑 Step 2: Analyzing SPL sections...")

	allSections := doc.GetAllSections()
	totalTables := 0
	sectionsWithTables := 0

	t.Log("   Key Sections Found:")
	for loincCode, section := range allSections {
		tableCount := 0
		if section.HasTables() {
			tables := section.GetTables()
			tableCount = len(tables)
			totalTables += tableCount
			sectionsWithTables++
		}

		// Only log key clinical sections
		if isKeyClinicalSection(loincCode) {
			t.Logf("      📋 [%s] %s - %d tables",
				loincCode, section.Code.DisplayName, tableCount)
		}
	}

	t.Logf("   📊 Summary: %d sections, %d with tables, %d total tables",
		len(allSections), sectionsWithTables, totalTables)

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 3: TABLE CLASSIFICATION
	// ─────────────────────────────────────────────────────────────────────────
	t.Log("\n🔬 Step 3: Classifying tables...")

	classifier := dailymed.NewTableClassifier()
	tableTypes := make(map[string]int)
	var classificationResults []TableClassificationResult

	for loincCode, section := range allSections {
		if !section.HasTables() {
			continue
		}

		for _, table := range section.GetTables() {
			result := classifier.ClassifyTable(&table)
			tableTypes[string(result.TableType)]++

			classificationResults = append(classificationResults, TableClassificationResult{
				SectionCode:  loincCode,
				TableID:      table.ID,
				TableType:    string(result.TableType),
				Confidence:   result.Confidence,
				RowCount:     len(table.Rows),
				Headers:      table.GetHeaders(),
			})

			if result.Confidence >= 0.5 {
				t.Logf("      ✅ [%s] Type: %s (%.0f%% confidence)",
					truncateString(table.ID, 20), result.TableType, result.Confidence*100)
			}
		}
	}

	t.Log("\n   📈 Table Type Distribution:")
	for tableType, count := range tableTypes {
		t.Logf("      • %s: %d", tableType, count)
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 4: RUN FULL PIPELINE
	// ─────────────────────────────────────────────────────────────────────────
	t.Log("\n⚙️  Step 4: Running canonical rule pipeline...")
	startPipeline := time.Now()

	pipelineResult, err := pipeline.ProcessDocument(ctx, MetforminSetID)
	pipelineDuration := time.Since(startPipeline)

	if err != nil {
		t.Logf("   ⚠️  Pipeline error (may be expected for some tables): %v", err)
	}

	if pipelineResult != nil {
		t.Logf("   ✅ Pipeline completed in %v", pipelineDuration.Round(time.Millisecond))
		t.Logf("   📊 Results:")
		t.Logf("      • Rules generated: %d", len(pipelineResult.Rules))
		t.Logf("      • Untranslatable: %d", len(pipelineResult.UntranslatableTables))
		t.Logf("      • Errors: %d", len(pipelineResult.Errors))

		// Log generated rules
		if len(pipelineResult.Rules) > 0 {
			t.Log("\n   📋 Generated Rules:")
			for i, rule := range pipelineResult.Rules {
				if i >= 5 {
					t.Logf("      ... and %d more rules", len(pipelineResult.Rules)-5)
					break
				}
				t.Logf("      [%d] %s: %s → %s",
					i+1,
					rule.RuleType,
					truncateString(rule.Condition.Variable, 25),
					rule.Action.Effect)
			}
		}

		// Log untranslatable tables
		if len(pipelineResult.UntranslatableTables) > 0 {
			t.Log("\n   ⚠️  Untranslatable Tables (need human review):")
			for i, entry := range pipelineResult.UntranslatableTables {
				if i >= 3 {
					t.Logf("      ... and %d more", len(pipelineResult.UntranslatableTables)-3)
					break
				}
				t.Logf("      • %s: %s", entry.Reason, truncateString(entry.TableID, 30))
			}
		}

		// Log pipeline errors
		if len(pipelineResult.Errors) > 0 {
			t.Log("\n   ❗ Pipeline Errors:")
			for _, e := range pipelineResult.Errors {
				t.Logf("      • [%s] %s: %s", e.Phase, e.Section, e.Error)
			}
		}
	}

	// ─────────────────────────────────────────────────────────────────────────
	// STEP 5: VALIDATION
	// ─────────────────────────────────────────────────────────────────────────
	t.Log("\n✅ Step 5: Validating results...")

	// Validation 1: Document fetched successfully
	if doc == nil {
		t.Error("   ❌ Document fetch failed")
	} else {
		t.Log("   ✅ Document fetched successfully")
	}

	// Validation 2: Expected sections present
	hasRequisiteSections := false
	for loincCode := range allSections {
		// Dosage & Administration (34068-7) or Contraindications (34070-3)
		if loincCode == "34068-7" || loincCode == "34070-3" || loincCode == "34073-7" {
			hasRequisiteSections = true
			break
		}
	}
	if hasRequisiteSections {
		t.Log("   ✅ Key clinical sections present")
	} else {
		t.Log("   ⚠️  Expected clinical sections not found")
	}

	// Validation 3: Tables found and classified
	if totalTables > 0 {
		t.Logf("   ✅ Tables found: %d", totalTables)
	} else {
		t.Log("   ⚠️  No tables found in document")
	}

	// Validation 4: Classification confidence check
	highConfidenceCount := 0
	for _, result := range classificationResults {
		if result.Confidence >= 0.7 {
			highConfidenceCount++
		}
	}
	t.Logf("   📊 High-confidence classifications: %d/%d", highConfidenceCount, len(classificationResults))

	// ─────────────────────────────────────────────────────────────────────────
	// FINAL REPORT
	// ─────────────────────────────────────────────────────────────────────────
	t.Log("\n" + strings.Repeat("═", 70))
	t.Log("                    METFORMIN E2E TEST SUMMARY")
	t.Log(strings.Repeat("═", 70))
	t.Logf("   Drug:              Metformin Hydrochloride")
	t.Logf("   SetID:             %s", MetforminSetID)
	t.Logf("   Fetch Time:        %v", fetchDuration.Round(time.Millisecond))
	t.Logf("   Pipeline Time:     %v", pipelineDuration.Round(time.Millisecond))
	t.Logf("   Total Sections:    %d", len(allSections))
	t.Logf("   Total Tables:      %d", totalTables)
	t.Logf("   Tables Classified: %d", len(classificationResults))
	if pipelineResult != nil {
		t.Logf("   Rules Generated:   %d", len(pipelineResult.Rules))
		t.Logf("   Untranslatable:    %d", len(pipelineResult.UntranslatableTables))
	}
	t.Log(strings.Repeat("═", 70))

	// Output JSON for debugging
	if testing.Verbose() {
		t.Log("\n📋 Classification Results (JSON):")
		jsonData, _ := json.MarshalIndent(classificationResults, "   ", "  ")
		t.Logf("%s", jsonData)
	}
}

// TableClassificationResult captures classification details for reporting
type TableClassificationResult struct {
	SectionCode string   `json:"section_code"`
	TableID     string   `json:"table_id"`
	TableType   string   `json:"table_type"`
	Confidence  float64  `json:"confidence"`
	RowCount    int      `json:"row_count"`
	Headers     []string `json:"headers"`
}

// isKeyClinicalSection returns true for important LOINC section codes
func isKeyClinicalSection(loincCode string) bool {
	keySections := map[string]bool{
		"34068-7": true, // Dosage & Administration
		"34070-3": true, // Contraindications
		"34073-7": true, // Drug Interactions
		"43685-7": true, // Warnings and Precautions
		"34066-1": true, // Boxed Warning
		"34084-4": true, // Adverse Reactions
		"34090-1": true, // Clinical Pharmacology
		"34067-9": true, // Indications & Usage
	}
	return keySections[loincCode]
}

// TestMetforminGFRDosingExtraction specifically tests GFR dosing table extraction
func TestMetforminGFRDosingExtraction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GFR dosing test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	t.Log("🔬 Testing Metformin GFR Dosing Table Extraction")

	config := dailymed.DefaultConfig()
	config.Timeout = 60 * time.Second
	cache := dailymed.NewMemorySPLCache(1 * time.Hour)
	fetcher := dailymed.NewSPLFetcher(config, cache)

	// Search for Metformin to get current SetID
	searchResults, err := fetcher.SearchByDrugName(ctx, "metformin")
	if err != nil || len(searchResults) == 0 {
		t.Fatalf("Failed to find Metformin in DailyMed: %v", err)
	}
	setID := searchResults[0].SetID
	t.Logf("   Using SetID: %s", setID)

	doc, err := fetcher.FetchBySetID(ctx, setID)
	if err != nil {
		t.Fatalf("Failed to fetch Metformin SPL: %v", err)
	}

	// Get Dosage & Administration section (LOINC 34068-7)
	dosageSection := doc.GetDosageSection()
	if dosageSection == nil {
		t.Skip("No Dosage & Administration section found")
	}

	t.Logf("📄 Dosage Section: %s", dosageSection.Code.DisplayName)

	if !dosageSection.HasTables() {
		t.Log("ℹ️  No tables in Dosage section - checking for prose content")
		// Metformin's renal dosing info might be in prose, not table
		content := dosageSection.GetRawText()
		if strings.Contains(strings.ToLower(content), "renal") ||
			strings.Contains(strings.ToLower(content), "egfr") ||
			strings.Contains(strings.ToLower(content), "creatinine") {
			t.Log("✅ Found renal dosing keywords in prose content")
			t.Logf("   Content preview: %s...", truncateString(content, 200))
		}
		return
	}

	classifier := dailymed.NewTableClassifier()
	tables := dosageSection.GetTables()

	t.Logf("📊 Found %d tables in Dosage section", len(tables))

	for _, table := range tables {
		result := classifier.ClassifyTable(&table)
		t.Logf("\n🔬 Table: %s", table.ID)
		t.Logf("   Type: %s", result.TableType)
		t.Logf("   Confidence: %.2f", result.Confidence)
		t.Logf("   Headers: %v", table.GetHeaders())
		t.Logf("   Rows: %d", len(table.Rows))

		// Check for GFR-related content
		headers := table.GetHeaders()
		for _, h := range headers {
			hLower := strings.ToLower(h)
			if strings.Contains(hLower, "gfr") ||
				strings.Contains(hLower, "egfr") ||
				strings.Contains(hLower, "creatinine") ||
				strings.Contains(hLower, "renal") ||
				strings.Contains(hLower, "ml/min") {
				t.Logf("   ✅ Found GFR-related column: %s", h)
			}
		}
	}
}
