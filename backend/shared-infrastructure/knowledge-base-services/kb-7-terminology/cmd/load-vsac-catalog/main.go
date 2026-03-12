// Package main - VSAC Semantic Catalog Loader
//
// Loads ALL 23,706 ValueSets from VSAC JSON source files into the value_sets table.
// This implements the REVISED KB-7 ARCHITECTURE (v2):
//
//   TIER 1: Full semantic catalog (23,706 from VSAC)
//   TIER 1.5: Canonical subset (~75-100 marked is_canonical)
//   TIER 2: Precomputed codes (5.3M, unchanged)
//
// Usage:
//   go run cmd/load-vsac-catalog/main.go
//
// Source: data/ontoserver-valuesets/definitions/*.json
// Target: value_sets table in PostgreSQL

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// FHIRValueSet represents a FHIR R4 ValueSet resource
type FHIRValueSet struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id"`
	URL          string `json:"url"`
	Version      string `json:"version"`
	Name         string `json:"name"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	Publisher    string `json:"publisher"`
	Description  string `json:"description"`
	Compose      *struct {
		Include []struct {
			System string `json:"system"`
		} `json:"include"`
	} `json:"compose"`
	Expansion *struct {
		Contains []struct {
			System string `json:"system"`
		} `json:"contains"`
	} `json:"expansion"`
}

// LoadStats tracks loading progress
type LoadStats struct {
	TotalFiles     int
	Processed      int
	Inserted       int
	Updated        int
	Skipped        int
	Errors         int
	StartTime      time.Time
	CategoryCounts map[string]int
}

// CanonicalValueSets defines the ~75-100 canonical ValueSets for ICU Intelligence, Safety
var CanonicalValueSets = map[string]bool{
	// Cardiovascular conditions
	"Essential Hypertension":              true,
	"Hypertensive Disorders":              true,
	"Heart Failure":                       true,
	"Coronary Artery Disease":             true,
	"Atrial Fibrillation":                 true,
	"Myocardial Infarction":               true,
	"Acute Coronary Syndrome":             true,

	// Diabetes
	"Diabetes":                            true,
	"Type 2 Diabetes Mellitus":            true,
	"Type 1 Diabetes Mellitus":            true,
	"Diabetic Nephropathy":                true,
	"Diabetic Retinopathy":                true,

	// Renal
	"Chronic Kidney Disease":              true,
	"Acute Kidney Injury":                 true,
	"End Stage Renal Disease":             true,

	// Respiratory
	"COPD":                                true,
	"Asthma":                              true,
	"Pneumonia":                           true,
	"Acute Respiratory Distress":          true,

	// Infectious
	"Sepsis":                              true,
	"Septic Shock":                        true,
	"Bacterial Infection":                 true,

	// Neurological
	"Stroke":                              true,
	"Ischemic Stroke":                     true,
	"Hemorrhagic Stroke":                  true,

	// Medications - ACE Inhibitors
	"ACE Inhibitors":                      true,
	"Lisinopril":                          true,
	"Enalapril":                           true,
	"Ramipril":                            true,

	// Medications - ARBs
	"ARBs":                                true,
	"Losartan":                            true,
	"Valsartan":                           true,

	// Medications - Beta Blockers
	"Beta Blockers":                       true,
	"Metoprolol":                          true,
	"Carvedilol":                          true,

	// Medications - Diabetes
	"Metformin":                           true,
	"Insulin":                             true,
	"SGLT2 Inhibitors":                    true,
	"GLP1 Receptor Agonists":              true,

	// Medications - Anticoagulants
	"Anticoagulants":                      true,
	"Warfarin":                            true,
	"DOACs":                               true,
	"Heparin":                             true,

	// Medications - Diuretics
	"Diuretics":                           true,
	"Loop Diuretics":                      true,
	"Thiazide Diuretics":                  true,

	// Medications - NSAIDs (safety)
	"NSAIDs":                              true,

	// Labs
	"HbA1c":                               true,
	"Creatinine":                          true,
	"eGFR":                                true,
	"Potassium":                           true,
	"Sodium":                              true,
	"BNP":                                 true,
	"Troponin":                            true,
	"Lactate":                             true,
	"Procalcitonin":                       true,
	"INR":                                 true,
	"LDL Cholesterol":                     true,
	"Blood Glucose":                       true,

	// Vital Signs
	"Blood Pressure":                      true,
	"Heart Rate":                          true,
	"Respiratory Rate":                    true,
	"Oxygen Saturation":                   true,
	"Temperature":                         true,
}

func main() {
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println("           VSAC SEMANTIC CATALOG LOADER - KB-7 Architecture v2")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println()

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://kb7_user:kb7_secure_password@localhost:5433/kb7_terminology?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		fmt.Printf("❌ Database ping failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Connected to PostgreSQL database")

	// Find definitions directory
	definitionsDir := findDefinitionsDir()
	if definitionsDir == "" {
		fmt.Println("❌ Could not find ontoserver-valuesets/definitions directory")
		os.Exit(1)
	}
	fmt.Printf("📁 Source directory: %s\n", definitionsDir)

	// Load all ValueSets
	stats := loadAllValueSets(db, definitionsDir)

	// Print summary
	printSummary(stats)
}

func findDefinitionsDir() string {
	// Try relative paths from different working directories
	candidates := []string{
		"data/ontoserver-valuesets/definitions",
		"../data/ontoserver-valuesets/definitions",
		"../../data/ontoserver-valuesets/definitions",
		"/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/data/ontoserver-valuesets/definitions",
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return ""
}

func loadAllValueSets(db *sql.DB, definitionsDir string) *LoadStats {
	stats := &LoadStats{
		StartTime:      time.Now(),
		CategoryCounts: make(map[string]int),
	}

	// Count total files first
	files, err := os.ReadDir(definitionsDir)
	if err != nil {
		fmt.Printf("❌ Failed to read directory: %v\n", err)
		return stats
	}

	jsonFiles := []fs.DirEntry{}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".json") && f.Name() != "_summary.json" {
			jsonFiles = append(jsonFiles, f)
		}
	}
	stats.TotalFiles = len(jsonFiles)
	fmt.Printf("📊 Found %d ValueSet JSON files to process\n\n", stats.TotalFiles)

	// Prepare insert statement
	insertSQL := `
		INSERT INTO value_sets (id, url, oid, version, name, title, description, status, publisher, category, is_canonical, source)
		VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'vsac')
		ON CONFLICT (url) DO UPDATE SET
			oid = EXCLUDED.oid,
			name = EXCLUDED.name,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			is_canonical = EXCLUDED.is_canonical,
			updated_at = NOW()
	`

	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		fmt.Printf("❌ Failed to prepare statement: %v\n", err)
		return stats
	}
	defer stmt.Close()

	// Process files in batches for progress reporting
	batchSize := 1000
	for i, f := range jsonFiles {
		filePath := filepath.Join(definitionsDir, f.Name())
		vs, err := parseValueSetFile(filePath)
		if err != nil {
			stats.Errors++
			continue
		}

		// Determine category and canonical status
		category := inferCategory(vs)
		isCanonical := isCanonicalValueSet(vs.Name, vs.Title)

		// Extract OID from ID or URL
		oid := extractOID(vs.ID, vs.URL)

		// Use sensible defaults
		version := vs.Version
		if version == "" {
			version = "1.0"
		}
		status := vs.Status
		if status == "" {
			status = "active"
		}

		// Insert or update
		_, err = stmt.Exec(
			vs.URL,
			oid,
			version,
			vs.Name,
			vs.Title,
			vs.Description,
			status,
			vs.Publisher,
			category,
			isCanonical,
		)
		if err != nil {
			stats.Errors++
			if stats.Errors <= 5 {
				fmt.Printf("⚠️  Error inserting %s: %v\n", vs.Name, err)
			}
		} else {
			stats.Inserted++
			stats.CategoryCounts[category]++
		}

		stats.Processed++

		// Progress report
		if (i+1)%batchSize == 0 || i == len(jsonFiles)-1 {
			pct := float64(i+1) / float64(stats.TotalFiles) * 100
			fmt.Printf("⏳ Progress: %d/%d (%.1f%%) - Inserted: %d, Errors: %d\n",
				i+1, stats.TotalFiles, pct, stats.Inserted, stats.Errors)
		}
	}

	// Update OID in precomputed_valueset_codes for reverse lookup optimization
	fmt.Println("\n🔗 Linking precomputed_valueset_codes to value_sets OIDs...")
	updateOIDsInPrecomputed(db)

	return stats
}

func parseValueSetFile(filePath string) (*FHIRValueSet, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var vs FHIRValueSet
	if err := json.Unmarshal(data, &vs); err != nil {
		return nil, err
	}

	return &vs, nil
}

func extractOID(id, url string) string {
	// OID is typically the ID when it's a numeric string like "2.16.840.1..."
	if strings.HasPrefix(id, "2.16.") {
		return id
	}

	// Extract from URL like "http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1..."
	if strings.Contains(url, "/ValueSet/") {
		parts := strings.Split(url, "/ValueSet/")
		if len(parts) > 1 {
			oid := parts[1]
			if strings.HasPrefix(oid, "2.16.") {
				return oid
			}
		}
	}

	return ""
}

func inferCategory(vs *FHIRValueSet) string {
	// Check name and title for category hints
	nameLower := strings.ToLower(vs.Name + " " + vs.Title)

	// Get code system from compose or expansion
	codeSystem := ""
	if vs.Compose != nil && len(vs.Compose.Include) > 0 {
		codeSystem = vs.Compose.Include[0].System
	} else if vs.Expansion != nil && len(vs.Expansion.Contains) > 0 {
		codeSystem = vs.Expansion.Contains[0].System
	}

	// Category inference rules
	if strings.Contains(codeSystem, "rxnorm") ||
		strings.Contains(nameLower, "medication") ||
		strings.Contains(nameLower, "drug") ||
		strings.Contains(nameLower, "pharmaceutical") {
		return "medication"
	}

	if strings.Contains(codeSystem, "loinc") ||
		strings.Contains(nameLower, "lab") ||
		strings.Contains(nameLower, "test") ||
		strings.Contains(nameLower, "measurement") ||
		strings.Contains(nameLower, "observation") {
		return "lab"
	}

	if strings.Contains(nameLower, "procedure") ||
		strings.Contains(nameLower, "surgery") ||
		strings.Contains(nameLower, "intervention") {
		return "procedure"
	}

	if strings.Contains(codeSystem, "icd") ||
		strings.Contains(nameLower, "diagnosis") ||
		strings.Contains(nameLower, "condition") ||
		strings.Contains(nameLower, "disease") ||
		strings.Contains(nameLower, "disorder") ||
		strings.Contains(nameLower, "syndrome") {
		return "condition"
	}

	if strings.Contains(codeSystem, "snomed") {
		// SNOMED can be conditions, procedures, or findings
		if strings.Contains(nameLower, "finding") ||
			strings.Contains(nameLower, "clinical") {
			return "condition"
		}
	}

	// Administrative/other
	if strings.Contains(nameLower, "gender") ||
		strings.Contains(nameLower, "status") ||
		strings.Contains(nameLower, "administrative") {
		return "administrative"
	}

	return "other"
}

func isCanonicalValueSet(name, title string) bool {
	// Check against known canonical list
	if CanonicalValueSets[name] || CanonicalValueSets[title] {
		return true
	}

	// Also check partial matches for flexibility
	combined := strings.ToLower(name + " " + title)
	for canonical := range CanonicalValueSets {
		if strings.Contains(combined, strings.ToLower(canonical)) {
			return true
		}
	}

	return false
}

func updateOIDsInPrecomputed(db *sql.DB) {
	// Update valueset_oid in precomputed_valueset_codes based on valueset_url
	updateSQL := `
		UPDATE precomputed_valueset_codes pvc
		SET valueset_oid = vs.oid
		FROM value_sets vs
		WHERE pvc.valueset_url = vs.url
		  AND vs.oid IS NOT NULL
		  AND vs.oid != ''
		  AND (pvc.valueset_oid IS NULL OR pvc.valueset_oid = '')
	`

	result, err := db.Exec(updateSQL)
	if err != nil {
		fmt.Printf("⚠️  Error updating OIDs: %v\n", err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Updated %d rows with OID linkage\n", rowsAffected)
}

func printSummary(stats *LoadStats) {
	elapsed := time.Since(stats.StartTime)

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println("                         LOADING COMPLETE")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Printf("📊 Total Files:      %d\n", stats.TotalFiles)
	fmt.Printf("✅ Inserted/Updated: %d\n", stats.Inserted)
	fmt.Printf("❌ Errors:           %d\n", stats.Errors)
	fmt.Printf("⏱️  Duration:         %s\n", elapsed.Round(time.Millisecond))
	fmt.Println()
	fmt.Println("📂 Category Breakdown:")
	for category, count := range stats.CategoryCounts {
		fmt.Printf("   %-15s %d\n", category+":", count)
	}
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println("                    KB-7 ARCHITECTURE v2 READY")
	fmt.Println("═══════════════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Verify: SELECT COUNT(*) FROM value_sets;")
	fmt.Println("2. Test reverse lookup: SELECT * FROM get_valueset_memberships('314076', 'http://www.nlm.nih.gov/research/umls/rxnorm');")
	fmt.Println("3. Check canonical: SELECT name FROM value_sets WHERE is_canonical = TRUE;")
}
