// Package main provides the SPL FactStore Pipeline CLI.
//
// This CLI executes the 9-phase SPL pipeline from the FactStore Execution Runbook:
//   Phase A: Verify Spine (drug_master, source_documents, source_sections)
//   Phase B: Select Scope (target drugs)
//   Phase C: SPL Acquisition (fetch from DailyMed)
//   Phase D: LOINC Section Routing (authority routing)
//   Phase E-F: Table Extraction & Rule Generation
//   Phase G: DraftFact Creation (6 canonical fact types)
//   Phase H: Governance Handoff (KB-0)
//   Phase I: KB Projection (to KB-1, KB-4, KB-5, KB-6, KB-16)
//
// Usage:
//   go run main.go --drug metformin
//   go run main.go --rxcui 6809
//   go run main.go --all-initial  # Run all 10 high-value drugs
//   go run main.go --verify-only  # Just verify spine tables
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cardiofit/shared/factstore"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/sirupsen/logrus"
)

// CLI flags
var (
	// Drug selection
	drugName   = flag.String("drug", "", "Drug name to process (e.g., metformin)")
	rxcui      = flag.String("rxcui", "", "RxCUI to process (e.g., 6809)")
	allInitial = flag.Bool("all-initial", false, "Process all 10 initial high-value drugs")
	drugsFile  = flag.String("drugs-file", "", "Path to file with drug list (one per line: rxcui,name)")

	// Pipeline control
	verifyOnly    = flag.Bool("verify-only", false, "Only verify spine tables, don't process")
	skipFetch     = flag.Bool("skip-fetch", false, "Skip SPL fetch, use cached documents")
	skipLLM       = flag.Bool("skip-llm", false, "Skip LLM extraction, use table parsing only")
	skipProjection = flag.Bool("skip-projection", false, "Skip KB projection phase")

	// Configuration
	dbURL      = flag.String("db-url", "", "PostgreSQL connection URL (or use KB_DATABASE_URL env)")
	rxnavURL   = flag.String("rxnav-url", "http://localhost:4000/REST", "RxNav-in-a-Box URL")
	dailymedURL = flag.String("dailymed-url", "https://dailymed.nlm.nih.gov/dailymed", "DailyMed base URL")

	// MedDRA configuration (Phase 3 Issues 2+3)
	mrconsoPath  = flag.String("mrconso", "", "Path to UMLS MRCONSO.RRF file for MedDRA loading (or use MRCONSO_PATH env)")
	mrhierPath   = flag.String("mrhier", "", "Path to UMLS MRHIER.RRF file for PT→SOC hierarchy (or use MRHIER_PATH env)")
	meddraDBPath    = flag.String("meddra-db", "", "Path to persist MedDRA SQLite DB (optional, in-memory if empty)")
	meddraValueSet  = flag.String("meddra-valueset", "", "Path to KB7 MedDRA ValueSet JSON (alternative to --mrconso, or use MEDDRA_VALUESET_PATH env)")

	// LLM Fallback
	anthropicKey = flag.String("anthropic-api-key", "", "Anthropic API key for LLM fallback (or ANTHROPIC_API_KEY env)")
	llmBudget    = flag.Float64("llm-budget", 50.0, "Maximum USD to spend on LLM calls per run")

	// Thresholds
	autoApprove = flag.Float64("auto-approve", 2.0, "Auto-approve threshold (2.0 = disabled). All facts route to PENDING_REVIEW for pharmacist review.")
	reviewThreshold = flag.Float64("review-threshold", 0.65, "Review threshold (>=)")
	minTableConf = flag.Float64("min-table-conf", 0.70, "Minimum table classification confidence")

	// Output control
	verbose  = flag.Bool("verbose", false, "Enable verbose logging")
	jsonOut  = flag.Bool("json", false, "Output results as JSON")
	dryRun   = flag.Bool("dry-run", false, "Parse and extract without saving to database")

	// Organ impairment enrichment (KDIGO only — CPIC removed, it's pharmacogenomics)
	includeOrganImpairment = flag.Bool("include-organ-impairment", false, "Enable organ impairment enrichment from KDIGO rules")

	// KDIGO rules from MCP-RAG atomiser (V2: JSON file, not PDF directory)
	kdigoRulesPath = flag.String("kdigo-rules", "", "Path to KDIGO rules JSON file (from MCP-RAG atomiser)")

	// Concurrency
	maxConcurrent = flag.Int("max-concurrent", 3, "Maximum concurrent drug processing")
	batchSize     = flag.Int("batch-size", 10, "Batch size for database operations")
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

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Warn("Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Print banner
	printBanner(log)

	// Get database URL
	dbConnURL := *dbURL
	if dbConnURL == "" {
		dbConnURL = os.Getenv("KB_DATABASE_URL")
	}
	if dbConnURL == "" {
		dbConnURL = "postgres://kb_admin:kb_secure_pass_2024@localhost:5433/canonical_facts?sslmode=disable"
	}

	// Open database connection
	db, err := sql.Open("postgres", dbConnURL)
	if err != nil {
		log.WithError(err).Fatal("Failed to open database connection")
	}
	defer db.Close()

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}

	// Initialize repository
	repo := factstore.NewRepository(db, log.WithField("component", "cli"))

	// Resolve MRCONSO path from flag or env
	mrconsoFile := *mrconsoPath
	if mrconsoFile == "" {
		mrconsoFile = os.Getenv("MRCONSO_PATH")
	}

	// Resolve MRHIER path from flag or env
	mrhierFile := *mrhierPath
	if mrhierFile == "" {
		mrhierFile = os.Getenv("MRHIER_PATH")
	}

	// Resolve MedDRA ValueSet path from flag or env
	valueSetFile := *meddraValueSet
	if valueSetFile == "" {
		valueSetFile = os.Getenv("MEDDRA_VALUESET_PATH")
	}

	// Resolve Anthropic API key from flag or env
	apiKey := *anthropicKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	// Build configuration
	config := factstore.RunnerConfig{
		VerifySpine:            true,
		FDABaseURL:             *dailymedURL,
		RxNavURL:               *rxnavURL,
		EnableAuthorityRouting: true,
		MinTableConfidence:     *minTableConf,
		FactTypes:              factstore.AllFactTypes(),
		AutoApproveThreshold:   *autoApprove,
		ReviewThreshold:        *reviewThreshold,
		EnableLLM:              !*skipLLM,
		EnableKBProjection:     !*skipProjection,
		TargetKBs:              []string{"KB-1", "KB-4", "KB-5", "KB-6", "KB-16"},
		MaxConcurrent:          *maxConcurrent,
		BatchSize:              *batchSize,
		MRCONSOPath:            mrconsoFile,
		MRHIERPath:             mrhierFile,
		MedDRADBPath:           *meddraDBPath,
		MedDRAValueSetPath:     valueSetFile,
		AnthropicAPIKey:        apiKey,
		LLMBudgetUSD:           *llmBudget,
		IncludeOrganImpairment: *includeOrganImpairment,
		KDIGORulesPath:         *kdigoRulesPath,
	}

	// Determine target drugs
	config.TargetDrugs = getTargetDrugs(log)
	if len(config.TargetDrugs) == 0 && !*verifyOnly {
		log.Fatal("No drugs specified. Use --drug, --rxcui, --all-initial, or --drugs-file")
	}

	// Create pipeline runner
	runner := factstore.NewPipelineRunner(config, repo, log)

	// Verify-only mode
	if *verifyOnly {
		log.Info("Running spine verification only...")
		if err := runner.VerifySpine(ctx); err != nil {
			log.WithError(err).Fatal("Spine verification failed")
		}
		log.Info("✅ Spine verification passed - all tables exist")
		return
	}

	// Dry run mode
	if *dryRun {
		log.Warn("DRY RUN MODE - No data will be saved to database")
	}

	// Run the pipeline
	log.WithFields(logrus.Fields{
		"drugs":         len(config.TargetDrugs),
		"llm_enabled":   config.EnableLLM,
		"kb_projection": config.EnableKBProjection,
	}).Info("Starting SPL Pipeline execution")

	startTime := time.Now()
	result, err := runner.Run(ctx)
	if err != nil {
		log.WithError(err).Fatal("Pipeline execution failed")
	}

	// Print results
	printResults(result, time.Since(startTime), log)

	// JSON output if requested
	if *jsonOut {
		printJSONResults(result)
	}
}

func printBanner(log *logrus.Logger) {
	banner := `
╔═══════════════════════════════════════════════════════════════════════════════╗
║              SPL FACTSTORE PIPELINE - Clinical Knowledge OS                   ║
║                   9-Phase Execution from Runbook Specification                ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║  Phases:                                                                      ║
║    A. Verify Spine      │ B. Select Scope     │ C. SPL Acquisition           ║
║    D. LOINC Routing     │ E-F. Table Extract  │ G. DraftFact Creation        ║
║    H. Governance        │ I. KB Projection                                   ║
╠═══════════════════════════════════════════════════════════════════════════════╣
║  Fact Types: ORGAN_IMPAIRMENT, SAFETY_SIGNAL, REPRODUCTIVE_SAFETY,           ║
║              INTERACTION, FORMULARY, LAB_REFERENCE                           ║
╚═══════════════════════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
	log.WithFields(logrus.Fields{
		"dry_run":       *dryRun,
		"skip_llm":      *skipLLM,
		"auto_approve":  *autoApprove,
		"max_concurrent": *maxConcurrent,
	}).Info("Configuration loaded")
}

func getTargetDrugs(log *logrus.Logger) []factstore.TargetDrug {
	var drugs []factstore.TargetDrug

	// All initial drugs from runbook
	if *allInitial {
		drugs = factstore.InitialScopeDrugs()
		log.WithField("count", len(drugs)).Info("Using all 10 initial high-value drugs from runbook")
		return drugs
	}

	// Single drug by name
	if *drugName != "" {
		drugs = append(drugs, factstore.TargetDrug{
			DrugName: *drugName,
			Reason:   "CLI specified",
		})
		log.WithField("drug", *drugName).Info("Processing single drug by name")
	}

	// Single drug by RxCUI
	if *rxcui != "" {
		drugs = append(drugs, factstore.TargetDrug{
			RxCUI:  *rxcui,
			Reason: "CLI specified",
		})
		log.WithField("rxcui", *rxcui).Info("Processing single drug by RxCUI")
	}

	// Drugs from file
	if *drugsFile != "" {
		fileDrugs, err := loadDrugsFromFile(*drugsFile)
		if err != nil {
			log.WithError(err).Fatal("Failed to load drugs from file")
		}
		drugs = append(drugs, fileDrugs...)
		log.WithFields(logrus.Fields{
			"file":  *drugsFile,
			"count": len(fileDrugs),
		}).Info("Loaded drugs from file")
	}

	return drugs
}

func loadDrugsFromFile(path string) ([]factstore.TargetDrug, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read drugs file: %w", err)
	}

	var drugs []factstore.TargetDrug
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			drugs = append(drugs, factstore.TargetDrug{
				RxCUI:    strings.TrimSpace(parts[0]),
				DrugName: strings.TrimSpace(parts[1]),
				Reason:   "From file",
			})
		} else if len(parts) == 1 {
			drugs = append(drugs, factstore.TargetDrug{
				DrugName: strings.TrimSpace(parts[0]),
				Reason:   "From file",
			})
		}
	}

	return drugs, nil
}

func printResults(result *factstore.PipelineRun, duration time.Duration, log *logrus.Logger) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                          SPL PIPELINE EXECUTION SUMMARY                       ║")
	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════╣")

	// Phase results
	fmt.Println("║  Phase Results:                                                               ║")
	phaseOrder := []string{"A", "B", "C", "D", "E-F", "G", "H", "I"}
	for _, phaseID := range phaseOrder {
		if phase, ok := result.Phases[phaseID]; ok {
			status := "✅"
			if phase.Status == "FAILED" {
				status = "❌"
			} else if phase.Status == "SKIPPED" {
				status = "⏭️"
			}
			fmt.Printf("║    %s Phase %s: %-50s        ║\n", status, phase.Phase, truncate(phase.Name, 50))
		}
	}

	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════╣")

	// Statistics
	fmt.Println("║  Statistics:                                                                  ║")
	fmt.Printf("║    SPLs Fetched:         %-10d                                            ║\n", result.Metrics.SPLsFetched)
	fmt.Printf("║    Sections Routed:      %-10d                                            ║\n", result.Metrics.SectionsRouted)
	fmt.Printf("║    Tables Classified:    %-10d                                            ║\n", result.Metrics.TablesClassified)
	fmt.Printf("║    Facts Created:        %-10d                                            ║\n", result.Metrics.FactsCreated)

	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════╣")

	// Governance
	fmt.Println("║  Governance Status:                                                           ║")
	fmt.Printf("║    Auto-Approved:        %-10d                                            ║\n", result.Metrics.FactsAutoApproved)
	fmt.Printf("║    Pending Review:       %-10d                                            ║\n", result.Metrics.FactsPendingReview)
	fmt.Printf("║    Rejected:             %-10d                                            ║\n", result.Metrics.FactsRejected)

	fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════╣")

	// Facts by type
	if len(result.Metrics.FactsByType) > 0 {
		fmt.Println("║  Facts by Type:                                                               ║")
		for factType, count := range result.Metrics.FactsByType {
			fmt.Printf("║    %-25s: %-10d                                      ║\n", factType, count)
		}
		fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════╣")
	}

	// KB projection
	if len(result.Metrics.KBProjections) > 0 {
		fmt.Println("║  Facts Projected to KBs:                                                      ║")
		for kb, count := range result.Metrics.KBProjections {
			fmt.Printf("║    %-10s: %-10d                                                    ║\n", kb, count)
		}
		fmt.Println("╠═══════════════════════════════════════════════════════════════════════════════╣")
	}

	// Duration
	fmt.Printf("║  Total Duration: %-60s  ║\n", duration.Round(time.Millisecond).String())
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Log summary
	log.WithFields(logrus.Fields{
		"spls_fetched":  result.Metrics.SPLsFetched,
		"facts":         result.Metrics.FactsCreated,
		"auto_approved": result.Metrics.FactsAutoApproved,
		"pending":       result.Metrics.FactsPendingReview,
		"duration":      duration,
	}).Info("Pipeline execution complete")
}

func printJSONResults(result *factstore.PipelineRun) {
	// Simple JSON output for scripting
	fmt.Printf(`{"run_id":"%s","status":"%s","spls_fetched":%d,"facts":%d,"auto_approved":%d,"pending_review":%d}`,
		result.ID,
		result.Status,
		result.Metrics.SPLsFetched,
		result.Metrics.FactsCreated,
		result.Metrics.FactsAutoApproved,
		result.Metrics.FactsPendingReview,
	)
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
