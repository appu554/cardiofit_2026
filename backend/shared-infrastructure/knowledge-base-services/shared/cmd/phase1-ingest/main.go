// Package main provides the Phase 1 Data Ingestion CLI for KB-5, KB-6, and KB-16.
//
// PHASE 1: "Ship Value WITHOUT LLM"
// This CLI loads structured data sources that require NO LLM:
// - ONC High-Priority DDI (~1,200 pairs)
// - CMS Medicare Part D Formulary
// - LOINC Lab Reference Ranges with NHANES enrichment
//
// Usage:
//   go run main.go --source onc --file ./data/onc_ddi.csv
//   go run main.go --source cms --file ./data/cms_formulary.csv
//   go run main.go --source loinc --file ./data/loinc_labs.csv --nhanes ./data/nhanes.csv
//   go run main.go --all --data-dir ./data
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cardiofit/shared/extraction/etl"
	"github.com/sirupsen/logrus"
)

// CLI flags
var (
	source      = flag.String("source", "", "Data source to load: onc, cms, loinc, ohdsi")
	dataFile    = flag.String("file", "", "Path to data file")
	nhanesFile  = flag.String("nhanes", "", "Path to NHANES statistics file (for LOINC)")
	dataDir     = flag.String("data-dir", "./data", "Directory containing all data files")
	loadAll     = flag.Bool("all", false, "Load all Phase 1 sources")
	dryRun      = flag.Bool("dry-run", false, "Parse and validate without saving to database")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
	batchSize   = flag.Int("batch-size", 100, "Batch size for database operations")
	skipInvalid = flag.Bool("skip-invalid", true, "Skip invalid records instead of failing")
)

func main() {
	flag.Parse()

	// Setup logging
	log := logrus.New()
	if *verbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	ctx := context.Background()

	// Print banner
	printBanner(log)

	// Determine what to load
	if *loadAll {
		loadAllSources(ctx, log)
		return
	}

	if *source == "" {
		log.Fatal("Please specify --source (onc, cms, loinc, ohdsi) or --all")
	}

	switch *source {
	case "onc":
		loadONCDDI(ctx, log)
	case "cms":
		loadCMSFormulary(ctx, log)
	case "loinc":
		loadLOINCLabs(ctx, log)
	case "ohdsi":
		loadOHDSIDDI(ctx, log)
	default:
		log.Fatalf("Unknown source: %s. Valid options: onc, cms, loinc, ohdsi", *source)
	}
}

func printBanner(log *logrus.Logger) {
	banner := `
╔═══════════════════════════════════════════════════════════════════════════════╗
║                   PHASE 1 DATA INGESTION - Clinical Knowledge OS              ║
║                        "Ship Value WITHOUT LLM"                               ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║  Sources:                                                                     ║
║    • ONC High-Priority DDI (~1,200 pairs)     - Drug-Drug Interactions        ║
║    • CMS Medicare Part D Formulary            - Coverage & Tiers              ║
║    • LOINC Lab Ranges + NHANES                - Reference Values              ║
║    • OHDSI Athena DDI (~200K pairs)           - Expanded Coverage             ║
╚═══════════════════════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
	log.WithFields(logrus.Fields{
		"dry_run":      *dryRun,
		"batch_size":   *batchSize,
		"skip_invalid": *skipInvalid,
	}).Info("Configuration loaded")
}

func loadAllSources(ctx context.Context, log *logrus.Logger) {
	log.Info("Loading all Phase 1 data sources...")

	results := make(map[string]interface{})
	startTime := time.Now()

	// Load ONC DDI
	oncFile := filepath.Join(*dataDir, "onc_ddi.csv")
	if fileExists(oncFile) {
		*dataFile = oncFile
		results["onc"] = loadONCDDI(ctx, log)
	} else {
		log.WithField("file", oncFile).Warn("ONC DDI file not found, skipping")
	}

	// Load CMS Formulary
	cmsFile := filepath.Join(*dataDir, "cms_formulary.csv")
	if fileExists(cmsFile) {
		*dataFile = cmsFile
		results["cms"] = loadCMSFormulary(ctx, log)
	} else {
		log.WithField("file", cmsFile).Warn("CMS Formulary file not found, skipping")
	}

	// Load LOINC Labs
	loincFile := filepath.Join(*dataDir, "loinc_labs.csv")
	if fileExists(loincFile) {
		*dataFile = loincFile
		nhanesPath := filepath.Join(*dataDir, "nhanes.csv")
		if fileExists(nhanesPath) {
			*nhanesFile = nhanesPath
		}
		results["loinc"] = loadLOINCLabs(ctx, log)
	} else {
		log.WithField("file", loincFile).Warn("LOINC Labs file not found, skipping")
	}

	// Print summary
	totalDuration := time.Since(startTime)
	log.WithFields(logrus.Fields{
		"total_duration": totalDuration,
		"sources_loaded": len(results),
	}).Info("Phase 1 data ingestion complete")

	printSummary(results, log)
}

func loadONCDDI(ctx context.Context, log *logrus.Logger) *etl.LoadResult {
	log.Info("═══════════════════════════════════════════════════════════════")
	log.Info("Loading ONC High-Priority DDI Dataset")
	log.Info("═══════════════════════════════════════════════════════════════")

	if *dataFile == "" {
		*dataFile = filepath.Join(*dataDir, "onc_ddi.csv")
	}

	config := etl.ONCLoaderConfig{
		SourcePath:     *dataFile,
		AutoActivate:   true, // ONC is authoritative
		BatchSize:      *batchSize,
		ValidateRxCUIs: false, // Skip validation in dry run
		SkipInvalid:    *skipInvalid,
	}

	// Create loader (nil factStore for dry run)
	var factStore etl.FactStoreWriter
	if !*dryRun {
		// TODO: Initialize actual fact store connection
		log.Warn("Database connection not configured - running in dry-run mode")
	}

	loader := etl.NewONCDDILoader(factStore, nil, config, log.WithField("loader", "ONC"))

	// Load with bidirectional pairs
	result, err := loader.LoadWithReverse(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to load ONC DDI data")
		return nil
	}

	// Add dataset metadata
	result.DatasetMetadata = &etl.DatasetMetadata{
		Version:            "2024-Q4",
		ReleaseDate:        time.Now(),
		RecordCount:        result.TotalParsed,
		SourceOrganization: "ONC/HHS",
		DownloadURL:        "https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction",
		DownloadedAt:       time.Now(),
	}

	log.WithFields(logrus.Fields{
		"total_parsed":     result.TotalParsed,
		"facts_created":    result.FactsCreated,
		"facts_skipped":    result.FactsSkipped,
		"validation_errors": result.ValidationErrs,
		"duration":         result.Duration,
	}).Info("ONC DDI load complete")

	return result
}

func loadCMSFormulary(ctx context.Context, log *logrus.Logger) *etl.CMSFormularyLoadResult {
	log.Info("═══════════════════════════════════════════════════════════════")
	log.Info("Loading CMS Medicare Part D Formulary")
	log.Info("═══════════════════════════════════════════════════════════════")

	if *dataFile == "" {
		*dataFile = filepath.Join(*dataDir, "cms_formulary.csv")
	}

	config := etl.CMSFormularyLoaderConfig{
		FormularyFilePath:   *dataFile,
		EffectiveYear:       2024,
		IncludeNonFormulary: false,
	}

	loader := etl.NewCMSFormularyLoader(config)

	result, err := loader.Load(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to load CMS Formulary data")
		return nil
	}

	log.WithFields(logrus.Fields{
		"total_processed":  result.TotalRowsProcessed,
		"entries_loaded":   result.EntriesLoaded,
		"plans_loaded":     result.PlansLoaded,
		"unique_rxcuis":    result.UniqueRxCUIs,
		"duration":         result.LoadDuration,
	}).Info("CMS Formulary load complete")

	return result
}

func loadLOINCLabs(ctx context.Context, log *logrus.Logger) *etl.LOINCLabLoadResult {
	log.Info("═══════════════════════════════════════════════════════════════")
	log.Info("Loading LOINC Lab Reference Ranges")
	log.Info("═══════════════════════════════════════════════════════════════")

	if *dataFile == "" {
		*dataFile = filepath.Join(*dataDir, "loinc_labs.csv")
	}

	config := etl.LOINCLabLoaderConfig{
		ReferenceRangeFilePath: *dataFile,
		NHANESFilePath:         *nhanesFile,
		IncludeDeprecated:      false,
	}

	loader := etl.NewLOINCLabLoader(config)

	result, err := loader.Load(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to load LOINC Labs data")
		return nil
	}

	log.WithFields(logrus.Fields{
		"total_processed":   result.TotalRowsProcessed,
		"ranges_loaded":     result.RangesLoaded,
		"unique_loinc":      result.UniqueLOINCCodes,
		"nhanes_enriched":   result.NHANESEnriched,
		"monitoring_loaded": result.MonitoringLoaded,
		"duration":          result.LoadDuration,
	}).Info("LOINC Labs load complete")

	// Print category distribution
	log.Info("Category Distribution:")
	for cat, count := range result.CategoryDistribution {
		log.WithFields(logrus.Fields{
			"category": cat,
			"count":    count,
		}).Debug("  ")
	}

	return result
}

func loadOHDSIDDI(ctx context.Context, log *logrus.Logger) *etl.OHDSILoadResult {
	log.Info("═══════════════════════════════════════════════════════════════")
	log.Info("Loading OHDSI Athena DDI (Expanded Coverage)")
	log.Info("═══════════════════════════════════════════════════════════════")

	conceptFile := filepath.Join(*dataDir, "ohdsi", "CONCEPT.csv")
	relationshipFile := filepath.Join(*dataDir, "ohdsi", "CONCEPT_RELATIONSHIP.csv")

	if !fileExists(conceptFile) || !fileExists(relationshipFile) {
		log.WithFields(logrus.Fields{
			"concept_file":      conceptFile,
			"relationship_file": relationshipFile,
		}).Warn("OHDSI Athena files not found, skipping")
		return nil
	}

	config := etl.OHDSIDDILoaderConfig{
		ConceptFilePath:      conceptFile,
		RelationshipFilePath: relationshipFile,
		OnlyStandardConcepts: true,
		IncludeExpired:       false,
	}

	loader := etl.NewOHDSIDDILoader(config)

	result, err := loader.Load(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to load OHDSI Athena data")
		return nil
	}

	log.WithFields(logrus.Fields{
		"total_concepts":   result.TotalConceptsLoaded,
		"drug_concepts":    result.DrugConceptsLoaded,
		"rxcui_mappings":   result.RxCUIMappings,
		"interactions":     result.InteractionsExtracted,
		"duration":         result.LoadDuration,
	}).Info("OHDSI Athena load complete")

	return result
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func printSummary(results map[string]interface{}, log *logrus.Logger) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                           PHASE 1 INGESTION SUMMARY                           ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════╣")

	for source, result := range results {
		if result == nil {
			fmt.Printf("║  %-12s: SKIPPED (file not found)                                       ║\n", source)
			continue
		}

		switch r := result.(type) {
		case *etl.LoadResult:
			fmt.Printf("║  %-12s: %d facts created, %d skipped, %v duration                    ║\n",
				source, r.FactsCreated, r.FactsSkipped, r.Duration.Round(time.Millisecond))
		case *etl.CMSFormularyLoadResult:
			fmt.Printf("║  %-12s: %d entries, %d plans, %d RxCUIs, %v duration                 ║\n",
				source, r.EntriesLoaded, r.PlansLoaded, r.UniqueRxCUIs, r.LoadDuration.Round(time.Millisecond))
		case *etl.LOINCLabLoadResult:
			fmt.Printf("║  %-12s: %d ranges, %d LOINC codes, %v duration                       ║\n",
				source, r.RangesLoaded, r.UniqueLOINCCodes, r.LoadDuration.Round(time.Millisecond))
		case *etl.OHDSILoadResult:
			fmt.Printf("║  %-12s: %d interactions, %d drugs, %v duration                       ║\n",
				source, r.InteractionsExtracted, r.DrugConceptsLoaded, r.LoadDuration.Round(time.Millisecond))
		}
	}

	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}
