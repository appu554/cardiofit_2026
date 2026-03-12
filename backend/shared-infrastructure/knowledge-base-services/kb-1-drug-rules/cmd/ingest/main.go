package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"kb-1-drug-rules/internal/config"
	"kb-1-drug-rules/internal/database"
	"kb-1-drug-rules/internal/rules"
	"kb-1-drug-rules/pkg/cache"
	"kb-1-drug-rules/pkg/ingestion"
)

func main() {
	// CLI flags
	source := flag.String("source", "fda", "Data source: fda, tga, cdsco, all")
	concurrency := flag.Int("concurrency", 0, "Number of parallel workers (0 = use config default)")
	dryRun := flag.Bool("dry-run", false, "Dry run without database writes (not yet implemented)")
	drugName := flag.String("drug", "", "Ingest specific drug by name")
	setID := flag.String("setid", "", "Ingest specific drug by SetID")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	healthCheck := flag.Bool("health", false, "Check health of dependent services")
	stats := flag.Bool("stats", false, "Show repository statistics")

	// Phase-A and Approval Workflow flags (CTO/CMO recommended)
	phaseA := flag.Bool("phase-a", false, "⚕️ RECOMMENDED: Ingest only Phase-A high-risk drugs (~200 drugs)")
	approvalStats := flag.Bool("approval-stats", false, "Show approval workflow statistics")
	pendingReview := flag.Bool("pending", false, "Show drugs pending pharmacist review")

	flag.Parse()

	// Initialize logger
	log := logrus.New()
	if *verbose {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	dbConfig := database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
	}

	logger := log.WithField("app", "kb1-ingest")
	db, err := database.Connect(dbConfig, logger)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Redis cache (optional)
	var redisCache *cache.RedisCache
	if cfg.Redis.Enabled {
		redisConfig := cache.Config{
			Host:        cfg.Redis.Host,
			Port:        cfg.Redis.Port,
			Password:    cfg.Redis.Password,
			DB:          cfg.Redis.DB,
			MaxRetries:  cfg.Redis.MaxRetries,
			PoolSize:    cfg.Redis.PoolSize,
			DialTimeout: cfg.Redis.DialTimeout,
			ReadTimeout: cfg.Redis.ReadTimeout,
		}
		redisCache, err = cache.NewRedisCache(redisConfig, logger)
		if err != nil {
			log.Warnf("Redis cache not available: %v (continuing without cache)", err)
			redisCache = nil
		} else {
			defer redisCache.Close()
		}
	}

	// Initialize repository
	var cacheInterface rules.Cache
	if redisCache != nil {
		cacheInterface = redisCache
	}
	repo := rules.NewRepository(db.DB, cacheInterface, logger)

	// Initialize ingestion engine
	engineConfig := ingestion.EngineConfig{
		RxNavURL:    cfg.RxNav.BaseURL,  // Direct RxNav API (rxnav-in-a-box)
		Concurrency: cfg.Ingestion.Concurrency,
		BatchSize:   cfg.Ingestion.BatchSize,
		RateLimitMs: cfg.Ingestion.RateLimitMs,
		FDABaseURL:  cfg.Ingestion.FDABaseURL,
	}
	if *concurrency > 0 {
		engineConfig.Concurrency = *concurrency
	}

	engine := ingestion.NewEngine(repo, engineConfig, logger)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("Received shutdown signal, cancelling...")
		cancel()
	}()

	// Handle different modes
	if *healthCheck {
		runHealthCheck(ctx, engine, logger)
		return
	}

	if *stats {
		showStats(ctx, engine, logger)
		return
	}

	if *approvalStats {
		showApprovalStats(ctx, repo, logger)
		return
	}

	if *pendingReview {
		showPendingReview(ctx, repo, logger)
		return
	}

	if *dryRun {
		log.Warn("Dry run mode not yet implemented")
	}

	// Targeted ingestion by drug name
	if *drugName != "" {
		runTargetedIngestion(ctx, engine, *drugName, logger)
		return
	}

	// Targeted ingestion by SetID
	if *setID != "" {
		runSetIDIngestion(ctx, engine, *setID, logger)
		return
	}

	// Phase-A ingestion (CTO/CMO RECOMMENDED)
	if *phaseA {
		runPhaseAIngestion(ctx, engine, logger)
		return
	}

	// Full ingestion - WARN about Phase-A
	runFullIngestion(ctx, engine, *source, logger)
}

func runHealthCheck(ctx context.Context, engine *ingestion.Engine, log *logrus.Entry) {
	fmt.Println("\n=== KB-1 Ingestion Health Check ===\n")

	results := engine.Health(ctx)

	allHealthy := true
	for service, err := range results {
		status := "✅ Healthy"
		if err != nil {
			status = fmt.Sprintf("❌ Unhealthy: %v", err)
			allHealthy = false
		}
		fmt.Printf("  %-10s: %s\n", service, status)
	}

	fmt.Println()
	if allHealthy {
		fmt.Println("All services healthy!")
		os.Exit(0)
	} else {
		fmt.Println("Some services are unhealthy!")
		os.Exit(1)
	}
}

func showStats(ctx context.Context, engine *ingestion.Engine, log *logrus.Entry) {
	fmt.Println("\n=== KB-1 Repository Statistics ===\n")

	stats, err := engine.GetIngestionStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}

	fmt.Printf("  Total Drugs:     %d\n", stats.TotalDrugs)
	fmt.Printf("  US (FDA):        %d\n", stats.USCount)
	fmt.Printf("  AU (TGA):        %d\n", stats.AUCount)
	fmt.Printf("  IN (CDSCO):      %d\n", stats.INCount)
	fmt.Printf("  High Alert:      %d\n", stats.HighAlertCount)
	fmt.Printf("  Black Box:       %d\n", stats.BlackBoxCount)
	if stats.LastIngestion != nil {
		fmt.Printf("  Last Ingestion:  %s\n", stats.LastIngestion.Format(time.RFC3339))
	}
	fmt.Println()
}

func runTargetedIngestion(ctx context.Context, engine *ingestion.Engine, drugName string, log *logrus.Entry) {
	fmt.Printf("\n=== KB-1 Targeted Ingestion ===\n")
	fmt.Printf("Drug Name: %s\n\n", drugName)

	result, err := engine.SearchAndIngestDrug(ctx, drugName, "CLI-targeted")
	if err != nil {
		log.Fatalf("Targeted ingestion failed: %v", err)
	}

	printResult(result)
}

func runSetIDIngestion(ctx context.Context, engine *ingestion.Engine, setID string, log *logrus.Entry) {
	fmt.Printf("\n=== KB-1 SetID Ingestion ===\n")
	fmt.Printf("SetID: %s\n\n", setID)

	run, err := engine.RunIncrementalFDAIngestion(ctx, []string{setID}, "CLI-setid")
	if err != nil {
		log.Fatalf("SetID ingestion failed: %v", err)
	}

	printRunSummary(run)
}

func runFullIngestion(ctx context.Context, engine *ingestion.Engine, source string, log *logrus.Entry) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ⚠️  FULL INGESTION WARNING                     ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  You are about to ingest 40,000+ drugs into DRAFT status.        ║")
	fmt.Println("║                                                                  ║")
	fmt.Println("║  CTO/CMO RECOMMENDATION: Use --phase-a flag instead!             ║")
	fmt.Println("║  Phase-A ingests ~200 high-risk drugs (80-90% clinical risk)     ║")
	fmt.Println("║  This is the recommended path for initial deployment.            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\nSource: %s\n\n", source)

	var run *ingestion.IngestionRun
	var err error

	switch source {
	case "fda":
		fmt.Println("Starting FDA DailyMed ingestion...")
		fmt.Println("This may take several hours for full formulary (~40,000+ drugs)")
		fmt.Println("All rules will be ingested in DRAFT status for pharmacist review")
		fmt.Println("Press Ctrl+C to cancel\n")
		run, err = engine.RunFDAIngestion(ctx, "CLI")
	case "tga":
		log.Fatal("TGA ingestion not yet implemented")
	case "cdsco":
		log.Fatal("CDSCO ingestion not yet implemented")
	case "all":
		fmt.Println("Starting FDA ingestion (other sources not yet implemented)...")
		run, err = engine.RunFDAIngestion(ctx, "CLI")
	default:
		log.Fatalf("Unknown source: %s", source)
	}

	if err != nil {
		log.Fatalf("Ingestion failed: %v", err)
	}

	printRunSummary(run)
}

func printResult(result *ingestion.DrugProcessResult) {
	fmt.Println("Result:")
	fmt.Printf("  SetID:      %s\n", result.SetID)
	fmt.Printf("  Drug Name:  %s\n", result.DrugName)
	fmt.Printf("  RxNorm:     %s\n", result.RxNormCode)
	fmt.Printf("  Action:     %s\n", result.Action)
	if result.Error != "" {
		fmt.Printf("  Error:      %s\n", result.Error)
	}
	fmt.Printf("  Duration:   %v\n", result.Duration)
	fmt.Println()
}

func printRunSummary(run *ingestion.IngestionRun) {
	fmt.Println("\n=== Ingestion Complete ===")
	fmt.Printf("Run ID:       %s\n", run.ID)
	fmt.Printf("Authority:    %s\n", run.Authority)
	fmt.Printf("Jurisdiction: %s\n", run.Jurisdiction)
	fmt.Printf("Status:       %s\n", run.Status)

	if run.CompletedAt != nil {
		fmt.Printf("Duration:     %v\n", run.CompletedAt.Sub(run.StartedAt))
	}

	fmt.Println("\nStatistics:")
	fmt.Printf("  Total Processed: %d\n", run.Stats.TotalProcessed)
	fmt.Printf("  Added:           %d\n", run.Stats.Added)
	fmt.Printf("  Updated:         %d\n", run.Stats.Updated)
	fmt.Printf("  Unchanged:       %d\n", run.Stats.Unchanged)
	fmt.Printf("  Skipped:         %d\n", run.Stats.Skipped)
	fmt.Printf("  Failed:          %d\n", run.Stats.Failed)

	// Quality metrics (CTO/CMO requirements)
	fmt.Println("\nQuality Metrics:")
	fmt.Printf("  🚨 Critical Risk:    %d\n", run.Stats.CriticalRisk)
	fmt.Printf("  ⚠️  High Risk:        %d\n", run.Stats.HighRisk)
	fmt.Printf("  📋 Requires Review:  %d\n", run.Stats.RequiresReview)
	fmt.Printf("  📊 Low Confidence:   %d\n", run.Stats.LowConfidence)
	fmt.Printf("  📈 Avg Confidence:   %d%%\n", run.Stats.AvgConfidence)

	if len(run.Errors) > 0 {
		fmt.Printf("\nErrors (%d):\n", len(run.Errors))
		maxErrors := 10
		for i, e := range run.Errors {
			if i >= maxErrors {
				fmt.Printf("  ... and %d more\n", len(run.Errors)-maxErrors)
				break
			}
			fmt.Printf("  - %s\n", e)
		}
	}

	fmt.Println()
	fmt.Println("⚕️  ALL RULES INGESTED IN DRAFT STATUS")
	fmt.Println("   Run --approval-stats to see pending reviews")
	fmt.Println("   Run --pending to see drugs awaiting pharmacist approval")
}

// =============================================================================
// PHASE-A INGESTION (CTO/CMO RECOMMENDED)
// =============================================================================

func runPhaseAIngestion(ctx context.Context, engine *ingestion.Engine, log *logrus.Entry) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║         ⚕️  PHASE-A FORMULARY INGESTION (RECOMMENDED)            ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Ingesting ~200 high-risk drugs that cover 80-90% clinical risk  ║")
	fmt.Println("║  Categories: Anticoagulants, Insulins, Opioids, ICU, Oncology    ║")
	fmt.Println("║  All rules will be in DRAFT status for pharmacist review         ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	result, err := engine.RunPhaseAIngestion(ctx, "CLI-PhaseA")
	if err != nil {
		log.Fatalf("Phase-A ingestion failed: %v", err)
	}

	// Print run summary
	printRunSummary(result.IngestionRun)

	// Print category breakdown
	fmt.Println("\n=== Category Breakdown ===")
	for catName, catStats := range result.CategoryResults {
		statusIcon := "✅"
		if catStats.FailedCount > 0 || len(catStats.MissingDrugs) > 0 {
			statusIcon = "⚠️"
		}
		fmt.Printf("%s %-20s [%s]: Found %d/%d, Ingested %d\n",
			statusIcon, catName, catStats.RiskLevel,
			catStats.FoundCount, catStats.TargetCount, catStats.IngestedCount)
	}

	// Print missing drugs
	if len(result.MissingDrugs) > 0 {
		fmt.Printf("\n⚠️  Missing Drugs (%d) - not found in FDA DailyMed:\n", len(result.MissingDrugs))
		for i, drug := range result.MissingDrugs {
			if i >= 20 {
				fmt.Printf("   ... and %d more\n", len(result.MissingDrugs)-20)
				break
			}
			fmt.Printf("   - %s\n", drug)
		}
	}

	fmt.Println("\n════════════════════════════════════════════════════════════════════")
	fmt.Println("✅ Phase-A ingestion complete!")
	fmt.Println("📋 Next step: Review pending rules with --pending flag")
	fmt.Println("⚕️  All rules require pharmacist approval before clinical use")
	fmt.Println("════════════════════════════════════════════════════════════════════")
}

// =============================================================================
// APPROVAL WORKFLOW VIEWS
// =============================================================================

func showApprovalStats(ctx context.Context, repo *rules.Repository, log *logrus.Entry) {
	fmt.Println("\n=== KB-1 Approval Workflow Statistics ===\n")

	stats, err := repo.GetApprovalStats(ctx)
	if err != nil {
		log.Fatalf("Failed to get approval stats: %v", err)
	}

	fmt.Println("📊 Rule Status Breakdown:")
	fmt.Printf("   Total Rules:      %d\n", stats.TotalRules)
	fmt.Printf("   📝 Draft:         %d\n", stats.DraftCount)
	fmt.Printf("   👁️  Reviewed:      %d\n", stats.ReviewedCount)
	fmt.Printf("   ✅ Active:        %d\n", stats.ActiveCount)
	fmt.Printf("   🗄️  Retired:       %d\n", stats.RetiredCount)

	fmt.Println("\n⚠️  Pending Review Queue:")
	fmt.Printf("   Total Pending:    %d\n", stats.PendingReview)
	fmt.Printf("   🚨 Critical Risk: %d (requires CMO sign-off)\n", stats.CriticalPending)
	fmt.Printf("   ⚠️  High Risk:     %d (requires pharmacist sign-off)\n", stats.HighPending)
	fmt.Printf("   📉 Low Confidence: %d (extraction quality < 50%%)\n", stats.LowConfidencePending)

	if stats.PendingReview > 0 {
		fmt.Println("\n💡 Recommendation:")
		fmt.Println("   Run --pending to see detailed review queue")
		fmt.Println("   CRITICAL risk drugs should be reviewed first")
	}

	fmt.Println()
}

func showPendingReview(ctx context.Context, repo *rules.Repository, log *logrus.Entry) {
	fmt.Println("\n=== KB-1 Pending Review Queue ===")
	fmt.Println("Rules awaiting pharmacist/CMO approval before clinical use\n")

	// Get pending reviews sorted by risk
	filter := rules.PendingReviewFilter{
		Limit: 50,
	}

	items, err := repo.GetPendingReviews(ctx, filter)
	if err != nil {
		log.Fatalf("Failed to get pending reviews: %v", err)
	}

	if len(items) == 0 {
		fmt.Println("✅ No rules pending review!")
		fmt.Println()
		return
	}

	fmt.Printf("Found %d rules pending review (showing top 50):\n\n", len(items))

	// Group by risk level for display
	var criticalItems, highItems, standardItems []rules.PendingReviewItem
	for _, item := range items {
		risk := "STANDARD"
		if item.RiskLevel != nil {
			risk = *item.RiskLevel
		}
		switch risk {
		case "CRITICAL":
			criticalItems = append(criticalItems, item)
		case "HIGH":
			highItems = append(highItems, item)
		default:
			standardItems = append(standardItems, item)
		}
	}

	if len(criticalItems) > 0 {
		fmt.Println("🚨 CRITICAL RISK (requires CMO + Pharmacist sign-off):")
		for _, item := range criticalItems {
			printPendingItem(item)
		}
		fmt.Println()
	}

	if len(highItems) > 0 {
		fmt.Println("⚠️  HIGH RISK (requires Pharmacist sign-off):")
		for _, item := range highItems {
			printPendingItem(item)
		}
		fmt.Println()
	}

	if len(standardItems) > 0 {
		fmt.Println("📋 STANDARD RISK:")
		maxShow := 10
		for i, item := range standardItems {
			if i >= maxShow {
				fmt.Printf("   ... and %d more\n", len(standardItems)-maxShow)
				break
			}
			printPendingItem(item)
		}
		fmt.Println()
	}

	fmt.Println("════════════════════════════════════════════════════════════════════")
	fmt.Println("💡 To approve rules, use the API or admin interface")
	fmt.Println("⚕️  CRITICAL risk drugs require explicit verification flag")
	fmt.Println("════════════════════════════════════════════════════════════════════")
}

func printPendingItem(item rules.PendingReviewItem) {
	confidence := 0
	if item.ExtractionConfidence != nil {
		confidence = *item.ExtractionConfidence
	}

	confidenceIcon := "📊"
	if confidence < 50 {
		confidenceIcon = "📉"
	}

	riskIcon := "📋"
	if item.RiskLevel != nil {
		switch *item.RiskLevel {
		case "CRITICAL":
			riskIcon = "🚨"
		case "HIGH":
			riskIcon = "⚠️"
		}
	}

	fmt.Printf("   %s %-30s [RxNorm: %s] %s %d%% confidence\n",
		riskIcon, item.DrugName, item.RxNormCode, confidenceIcon, confidence)
}
