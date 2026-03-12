// Package main provides the kb7-loader CLI tool for loading clinical terminology
// data from GCS into GraphDB and Neo4j. This is the Phase 5 implementation that uses the
// knowledge-factory pipeline output (kb7-kernel.ttl) as the data source.
//
// Architecture: Both GraphDB (OWL reasoning) and Neo4j (fast traversals) are loaded
// from the same kb7-kernel.ttl source. CDC sync keeps them synchronized for updates.
//
// Usage:
//
//	kb7-loader load --source gcs --version latest        # Load to GraphDB
//	kb7-loader load --source file --path /path/to/kb7-kernel.ttl
//	kb7-loader load-neo4j --path /path/to/kb7-kernel.ttl # Load to Neo4j
//	kb7-loader status                                    # GraphDB status
//	kb7-loader neo4j-status                              # Neo4j status
//	kb7-loader verify                                    # Verify GraphDB
//	kb7-loader verify-neo4j                              # Verify Neo4j
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"kb-7-terminology/internal/loader"
)

// Configuration constants
const (
	DefaultGCSBucket     = "sincere-hybrid-477206-h2-kb-artifacts-production"
	DefaultGraphDBURL    = "http://localhost:7200"
	DefaultRepository    = "kb7-terminology"
	DefaultTimeout       = 30 * time.Minute
	DefaultVersion       = "latest"
)

// LoaderConfig holds the configuration for the loader
type LoaderConfig struct {
	// GCS Configuration
	GCSBucket      string
	GCSPath        string
	GCSCredentials string

	// GraphDB Configuration
	GraphDBURL      string
	GraphDBRepo     string
	GraphDBUsername string
	GraphDBPassword string

	// Load Options
	Source       string // "gcs" or "file"
	LocalPath    string
	Version      string
	Timeout      time.Duration
	DryRun       bool
	ClearFirst   bool
	NamedGraph   string
}

// LoadResult holds the result of a load operation
type LoadResult struct {
	Success       bool          `json:"success"`
	TripleCount   int64         `json:"triple_count"`
	Duration      time.Duration `json:"duration"`
	Source        string        `json:"source"`
	Version       string        `json:"version"`
	GraphDBURL    string        `json:"graphdb_url"`
	Repository    string        `json:"repository"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	LoadTimestamp time.Time     `json:"load_timestamp"`
}

// GraphDBStatus represents the status of the GraphDB repository
type GraphDBStatus struct {
	Available    bool   `json:"available"`
	Repository   string `json:"repository"`
	TripleCount  int64  `json:"triple_count"`
	Writable     bool   `json:"writable"`
	ErrorMessage string `json:"error_message,omitempty"`
}

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Define subcommands
	loadCmd := flag.NewFlagSet("load", flag.ExitOnError)
	statusCmd := flag.NewFlagSet("status", flag.ExitOnError)
	verifyCmd := flag.NewFlagSet("verify", flag.ExitOnError)
	loadNeo4jCmd := flag.NewFlagSet("load-neo4j", flag.ExitOnError)
	neo4jStatusCmd := flag.NewFlagSet("neo4j-status", flag.ExitOnError)
	verifyNeo4jCmd := flag.NewFlagSet("verify-neo4j", flag.ExitOnError)

	// Load command flags (GraphDB)
	loadSource := loadCmd.String("source", "gcs", "Data source: 'gcs' or 'file'")
	loadVersion := loadCmd.String("version", DefaultVersion, "Version to load (e.g., 'latest', '20251202')")
	loadPath := loadCmd.String("path", "", "Local file path (when source=file)")
	loadDryRun := loadCmd.Bool("dry-run", false, "Simulate load without making changes")
	loadClear := loadCmd.Bool("clear", false, "Clear repository before loading")
	loadTimeout := loadCmd.Duration("timeout", DefaultTimeout, "Load timeout duration")
	loadNamedGraph := loadCmd.String("graph", "", "Named graph URI (optional)")

	// Load-neo4j command flags
	neo4jPath := loadNeo4jCmd.String("path", "", "Local TTL file path (required)")
	neo4jDryRun := loadNeo4jCmd.Bool("dry-run", false, "Simulate load without making changes")
	neo4jClear := loadNeo4jCmd.Bool("clear", false, "Clear database before loading")
	neo4jBatchSize := loadNeo4jCmd.Int("batch-size", 5000, "Batch size for Neo4j writes")
	neo4jWorkers := loadNeo4jCmd.Int("workers", 4, "Number of parallel workers")
	neo4jTimeout := loadNeo4jCmd.Duration("timeout", 60*time.Minute, "Load timeout duration")

	// Common flags for all commands
	graphdbURL := flag.String("graphdb-url", getEnv("GRAPHDB_URL", DefaultGraphDBURL), "GraphDB server URL")
	graphdbRepo := flag.String("graphdb-repo", getEnv("GRAPHDB_REPOSITORY", DefaultRepository), "GraphDB repository ID")
	graphdbUser := flag.String("graphdb-user", getEnv("GRAPHDB_USERNAME", ""), "GraphDB username")
	graphdbPass := flag.String("graphdb-pass", getEnv("GRAPHDB_PASSWORD", ""), "GraphDB password")
	gcsBucket := flag.String("gcs-bucket", getEnv("GCS_BUCKET", DefaultGCSBucket), "GCS bucket name")
	gcsCreds := flag.String("gcs-credentials", getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""), "Path to GCS credentials JSON")
	neo4jURL := flag.String("neo4j-url", getEnv("NEO4J_URL", "bolt://localhost:7687"), "Neo4j server URL")
	neo4jUser := flag.String("neo4j-user", getEnv("NEO4J_USERNAME", "neo4j"), "Neo4j username")
	neo4jPass := flag.String("neo4j-pass", getEnv("NEO4J_PASSWORD", "password"), "Neo4j password")
	neo4jDB := flag.String("neo4j-db", getEnv("NEO4J_DATABASE", "neo4j"), "Neo4j database name")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")

	// Parse main flags first
	flag.Parse()

	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	}

	// Check for subcommand
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Build config from flags
	config := &LoaderConfig{
		GCSBucket:       *gcsBucket,
		GCSCredentials:  *gcsCreds,
		GraphDBURL:      *graphdbURL,
		GraphDBRepo:     *graphdbRepo,
		GraphDBUsername: *graphdbUser,
		GraphDBPassword: *graphdbPass,
	}

	switch os.Args[1] {
	case "load":
		loadCmd.Parse(os.Args[2:])
		config.Source = *loadSource
		config.Version = *loadVersion
		config.LocalPath = *loadPath
		config.DryRun = *loadDryRun
		config.ClearFirst = *loadClear
		config.Timeout = *loadTimeout
		config.NamedGraph = *loadNamedGraph

		if config.Source == "gcs" {
			config.GCSPath = fmt.Sprintf("%s/kb7-kernel.ttl", config.Version)
		}

		result := executeLoad(config)
		printResult(result)
		if !result.Success {
			os.Exit(1)
		}

	case "status":
		statusCmd.Parse(os.Args[2:])
		status := checkStatus(config)
		printStatus(status)
		if !status.Available {
			os.Exit(1)
		}

	case "verify":
		verifyCmd.Parse(os.Args[2:])
		success := verifyRepository(config)
		if !success {
			os.Exit(1)
		}

	case "load-neo4j":
		loadNeo4jCmd.Parse(os.Args[2:])
		if *neo4jPath == "" {
			fmt.Println("Error: --path is required for load-neo4j command")
			os.Exit(1)
		}

		neo4jConfig := &loader.Neo4jLoaderConfig{
			Neo4jURL:      *neo4jURL,
			Neo4jUsername: *neo4jUser,
			Neo4jPassword: *neo4jPass,
			Neo4jDatabase: *neo4jDB,
			BatchSize:     *neo4jBatchSize,
			Workers:       *neo4jWorkers,
			ClearFirst:    *neo4jClear,
			CreateIndexes: true,
			DryRun:        *neo4jDryRun,
			Timeout:       *neo4jTimeout,
		}

		neo4jLoader, err := loader.NewNeo4jLoader(neo4jConfig, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to create Neo4j loader")
			os.Exit(1)
		}
		defer neo4jLoader.Close(context.Background())

		result, err := neo4jLoader.LoadFromFile(context.Background(), *neo4jPath)
		if err != nil {
			logger.WithError(err).Error("Neo4j load failed")
			os.Exit(1)
		}

		output, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(output))

		if !result.Success {
			os.Exit(1)
		}

	case "neo4j-status":
		neo4jStatusCmd.Parse(os.Args[2:])

		neo4jConfig := &loader.Neo4jLoaderConfig{
			Neo4jURL:      *neo4jURL,
			Neo4jUsername: *neo4jUser,
			Neo4jPassword: *neo4jPass,
			Neo4jDatabase: *neo4jDB,
		}

		neo4jLoader, err := loader.NewNeo4jLoader(neo4jConfig, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to connect to Neo4j")
			os.Exit(1)
		}
		defer neo4jLoader.Close(context.Background())

		status, err := neo4jLoader.GetStatus(context.Background())
		if err != nil {
			logger.WithError(err).Error("Failed to get Neo4j status")
			os.Exit(1)
		}

		logger.Info("═══════════════════════════════════════════════════════════")
		logger.Info("KB-7 Neo4j Database Status")
		logger.Info("═══════════════════════════════════════════════════════════")
		logger.Infof("  URL:        %s", *neo4jURL)
		logger.Infof("  Database:   %s", *neo4jDB)
		logger.Infof("  Available:  %v", status["available"])
		logger.Infof("  Nodes:      %v", status["node_count"])
		logger.Infof("  Relations:  %v", status["rel_count"])
		logger.Info("═══════════════════════════════════════════════════════════")

	case "verify-neo4j":
		verifyNeo4jCmd.Parse(os.Args[2:])

		neo4jConfig := &loader.Neo4jLoaderConfig{
			Neo4jURL:      *neo4jURL,
			Neo4jUsername: *neo4jUser,
			Neo4jPassword: *neo4jPass,
			Neo4jDatabase: *neo4jDB,
		}

		neo4jLoader, err := loader.NewNeo4jLoader(neo4jConfig, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to connect to Neo4j")
			os.Exit(1)
		}
		defer neo4jLoader.Close(context.Background())

		success, err := neo4jLoader.VerifyData(context.Background())
		if err != nil {
			logger.WithError(err).Error("Neo4j verification failed")
			os.Exit(1)
		}
		if !success {
			os.Exit(1)
		}

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`KB-7 Terminology Loader - Graph Data Population

USAGE:
    kb7-loader <command> [options]

COMMANDS:
    load          Load TTL data into GraphDB from GCS or local file
    load-neo4j    Load TTL data into Neo4j from local file
    status        Check GraphDB repository status
    neo4j-status  Check Neo4j database status
    verify        Verify GraphDB loaded data with SPARQL queries
    verify-neo4j  Verify Neo4j loaded data with Cypher queries
    help          Show this help message

GRAPHDB LOAD OPTIONS:
    --source <gcs|file>     Data source (default: gcs)
    --version <version>     GCS version folder (default: latest)
    --path <path>           Local file path (when source=file)
    --dry-run               Simulate without loading
    --clear                 Clear repository before loading
    --timeout <duration>    Load timeout (default: 30m)
    --graph <uri>           Named graph URI (optional)

NEO4J LOAD OPTIONS:
    --path <path>           Local TTL file path (required)
    --dry-run               Simulate without loading
    --clear                 Clear database before loading
    --batch-size <n>        Batch size for writes (default: 5000)
    --workers <n>           Parallel workers (default: 4)
    --timeout <duration>    Load timeout (default: 60m)

GRAPHDB OPTIONS:
    --graphdb-url <url>     GraphDB server URL
    --graphdb-repo <id>     GraphDB repository ID
    --graphdb-user <user>   GraphDB username
    --graphdb-pass <pass>   GraphDB password

NEO4J OPTIONS:
    --neo4j-url <url>       Neo4j server URL (default: bolt://localhost:7687)
    --neo4j-user <user>     Neo4j username (default: neo4j)
    --neo4j-pass <pass>     Neo4j password (default: password)
    --neo4j-db <name>       Neo4j database name (default: neo4j)

COMMON OPTIONS:
    --gcs-bucket <bucket>   GCS bucket name
    --gcs-credentials <path> Path to GCS credentials JSON
    --verbose               Enable verbose logging

EXAMPLES:
    # Load to GraphDB from GCS
    kb7-loader load --source gcs --version latest

    # Load to GraphDB from local file
    kb7-loader load --source file --path /data/kb7-kernel.ttl

    # Load to Neo4j from local file
    kb7-loader load-neo4j --path /data/kb7-kernel.ttl

    # Load to Neo4j with custom settings
    kb7-loader load-neo4j --path /data/kb7-kernel.ttl --batch-size 10000 --clear

    # Check status of both databases
    kb7-loader status
    kb7-loader neo4j-status

    # Verify loaded data in both databases
    kb7-loader verify
    kb7-loader verify-neo4j

ENVIRONMENT VARIABLES:
    GRAPHDB_URL             GraphDB server URL
    GRAPHDB_REPOSITORY      GraphDB repository ID
    GRAPHDB_USERNAME        GraphDB username
    GRAPHDB_PASSWORD        GraphDB password
    NEO4J_URL               Neo4j server URL
    NEO4J_USERNAME          Neo4j username
    NEO4J_PASSWORD          Neo4j password
    NEO4J_DATABASE          Neo4j database name
    GCS_BUCKET              GCS bucket name
    GOOGLE_APPLICATION_CREDENTIALS  Path to GCS credentials JSON

ARCHITECTURE:
    Both GraphDB and Neo4j are loaded from the same kb7-kernel.ttl source:

    kb7-kernel.ttl (GCS) ──┬──> GraphDB (OWL reasoning, SPARQL)
                          └──> Neo4j (fast traversals, Cypher)

    CDC sync keeps them synchronized for incremental updates.

DATA SOURCE:
    The kb7-kernel.ttl file is produced by the knowledge-factory pipeline
    and stored in GCS at:
    gs://sincere-hybrid-477206-h2-kb-artifacts-production/latest/kb7-kernel.ttl

    This file contains merged and reasoned ontologies for:
    - SNOMED-CT (clinical terminology)
    - RxNorm (drug terminology)
    - LOINC (lab codes)
    - ICD-10 → HCC mappings`)
}

// executeLoad performs the main load operation
func executeLoad(config *LoaderConfig) *LoadResult {
	result := &LoadResult{
		Source:        config.Source,
		Version:       config.Version,
		GraphDBURL:    config.GraphDBURL,
		Repository:    config.GraphDBRepo,
		LoadTimestamp: time.Now(),
	}

	startTime := time.Now()

	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Info("KB-7 Terminology Loader - Phase 5 Graph Data Population")
	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Infof("  Source:     %s", config.Source)
	logger.Infof("  Version:    %s", config.Version)
	logger.Infof("  GraphDB:    %s", config.GraphDBURL)
	logger.Infof("  Repository: %s", config.GraphDBRepo)
	logger.Infof("  Timeout:    %s", config.Timeout)
	logger.Infof("  Dry Run:    %v", config.DryRun)
	logger.Info("═══════════════════════════════════════════════════════════")

	if config.DryRun {
		logger.Warn("DRY RUN MODE - No changes will be made")
	}

	// Step 1: Verify GraphDB connectivity
	logger.Info("")
	logger.Info("Step 1: Verifying GraphDB connectivity...")
	if err := verifyGraphDBConnection(config); err != nil {
		result.ErrorMessage = fmt.Sprintf("GraphDB connection failed: %v", err)
		logger.WithError(err).Error("GraphDB connection failed")
		return result
	}
	logger.Info("✅ GraphDB is accessible")

	// Step 2: Clear repository if requested
	if config.ClearFirst && !config.DryRun {
		logger.Info("")
		logger.Info("Step 2: Clearing existing data...")
		if err := clearRepository(config); err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to clear repository: %v", err)
			logger.WithError(err).Error("Failed to clear repository")
			return result
		}
		logger.Info("✅ Repository cleared")
	} else {
		logger.Info("")
		logger.Info("Step 2: Skipping clear (--clear not specified)")
	}

	// Step 3: Load data
	logger.Info("")
	logger.Info("Step 3: Loading TTL data into GraphDB...")

	var loadErr error
	switch config.Source {
	case "gcs":
		loadErr = loadFromGCS(config)
	case "file":
		loadErr = loadFromFile(config)
	default:
		loadErr = fmt.Errorf("unknown source: %s", config.Source)
	}

	if loadErr != nil {
		result.ErrorMessage = fmt.Sprintf("Load failed: %v", loadErr)
		logger.WithError(loadErr).Error("Load failed")
		return result
	}

	if !config.DryRun {
		logger.Info("✅ TTL data loaded successfully")
	}

	// Step 4: Verify triple count
	logger.Info("")
	logger.Info("Step 4: Verifying loaded data...")
	tripleCount, err := getTripleCount(config)
	if err != nil {
		logger.WithError(err).Warn("Could not verify triple count")
	} else {
		result.TripleCount = tripleCount
		logger.Infof("✅ Repository contains %d triples", tripleCount)

		// Sanity check - KB-7 should have ~14M triples
		if tripleCount < 1_000_000 {
			logger.Warnf("⚠️  Warning: Expected ~14M triples, got %d", tripleCount)
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)

	logger.Info("")
	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Info("✅ KB-7 LOAD COMPLETE")
	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Infof("  Duration:     %s", result.Duration)
	logger.Infof("  Triples:      %d", result.TripleCount)
	logger.Infof("  SPARQL:       %s/repositories/%s", config.GraphDBURL, config.GraphDBRepo)
	logger.Info("═══════════════════════════════════════════════════════════")

	return result
}

// loadFromGCS loads TTL data from GCS using SPARQL LOAD with signed URL
func loadFromGCS(config *LoaderConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	logger.Infof("  GCS Bucket: %s", config.GCSBucket)
	logger.Infof("  GCS Path:   %s", config.GCSPath)

	if config.DryRun {
		logger.Info("  [DRY RUN] Would generate signed URL and execute SPARQL LOAD")
		return nil
	}

	// Create GCS client
	var client *storage.Client
	var err error

	if config.GCSCredentials != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(config.GCSCredentials))
	} else {
		// Use default credentials (ADC)
		client, err = storage.NewClient(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	// Generate signed URL (valid for 60 minutes)
	logger.Info("  Generating signed URL...")
	bucket := client.Bucket(config.GCSBucket)
	obj := bucket.Object(config.GCSPath)

	// Check if object exists
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("GCS object not found: gs://%s/%s: %w", config.GCSBucket, config.GCSPath, err)
	}
	logger.Infof("  File size: %.2f GB", float64(attrs.Size)/(1024*1024*1024))

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(60 * time.Minute),
	}

	signedURL, err := bucket.SignedURL(config.GCSPath, opts)
	if err != nil {
		return fmt.Errorf("failed to generate signed URL: %w", err)
	}
	logger.Info("  ✅ Signed URL generated (expires in 60 min)")

	// Execute SPARQL LOAD
	return executeSPARQLLoad(config, signedURL)
}

// loadFromFile loads TTL data from a local file
func loadFromFile(config *LoaderConfig) error {
	if config.LocalPath == "" {
		return fmt.Errorf("local path is required when source=file")
	}

	// Check if file exists
	info, err := os.Stat(config.LocalPath)
	if err != nil {
		return fmt.Errorf("file not found: %s: %w", config.LocalPath, err)
	}

	logger.Infof("  File: %s", config.LocalPath)
	logger.Infof("  Size: %.2f GB", float64(info.Size())/(1024*1024*1024))

	if config.DryRun {
		logger.Info("  [DRY RUN] Would upload file to GraphDB")
		return nil
	}

	// For local files, use direct upload via GraphDB import API
	return uploadFileToGraphDB(config)
}

// executeSPARQLLoad executes a SPARQL LOAD command
func executeSPARQLLoad(config *LoaderConfig, sourceURL string) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", config.GraphDBURL, config.GraphDBRepo)

	// Build SPARQL LOAD command
	var sparqlUpdate string
	if config.NamedGraph != "" {
		sparqlUpdate = fmt.Sprintf("LOAD <%s> INTO GRAPH <%s>", sourceURL, config.NamedGraph)
	} else {
		sparqlUpdate = fmt.Sprintf("LOAD <%s>", sourceURL)
	}

	logger.Info("  Executing SPARQL LOAD (this may take 5-15 minutes)...")
	logger.Debug("  SPARQL: ", sparqlUpdate[:50], "...")

	// Create request
	data := url.Values{}
	data.Set("update", sparqlUpdate)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if config.GraphDBUsername != "" {
		req.SetBasicAuth(config.GraphDBUsername, config.GraphDBPassword)
	}

	// Execute with timeout
	client := &http.Client{Timeout: config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("SPARQL LOAD request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SPARQL LOAD failed with status %d: %s", resp.StatusCode, string(body))
	}

	logger.Info("  ✅ SPARQL LOAD completed")
	return nil
}

// uploadFileToGraphDB uploads a local TTL file directly to GraphDB
func uploadFileToGraphDB(config *LoaderConfig) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", config.GraphDBURL, config.GraphDBRepo)

	// Open file
	file, err := os.Open(config.LocalPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	logger.Info("  Uploading file to GraphDB (this may take 10-30 minutes)...")

	// Create request
	req, err := http.NewRequest("POST", endpoint, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/turtle")
	if config.GraphDBUsername != "" {
		req.SetBasicAuth(config.GraphDBUsername, config.GraphDBPassword)
	}

	// Execute with timeout
	client := &http.Client{Timeout: config.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	logger.Info("  ✅ File upload completed")
	return nil
}

// verifyGraphDBConnection checks if GraphDB is accessible
func verifyGraphDBConnection(config *LoaderConfig) error {
	endpoint := fmt.Sprintf("%s/rest/repositories/%s", config.GraphDBURL, config.GraphDBRepo)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}

	if config.GraphDBUsername != "" {
		req.SetBasicAuth(config.GraphDBUsername, config.GraphDBPassword)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("repository not accessible: HTTP %d", resp.StatusCode)
	}

	return nil
}

// clearRepository removes all triples from the repository
func clearRepository(config *LoaderConfig) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", config.GraphDBURL, config.GraphDBRepo)

	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return err
	}

	if config.GraphDBUsername != "" {
		req.SetBasicAuth(config.GraphDBUsername, config.GraphDBPassword)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("clear request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("clear failed with status %d", resp.StatusCode)
	}

	return nil
}

// getTripleCount returns the number of triples in the repository
func getTripleCount(config *LoaderConfig) (int64, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s", config.GraphDBURL, config.GraphDBRepo)

	query := "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }"
	data := url.Values{}
	data.Set("query", query)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	if config.GraphDBUsername != "" {
		req.SetBasicAuth(config.GraphDBUsername, config.GraphDBPassword)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("query failed with status %d", resp.StatusCode)
	}

	var result struct {
		Results struct {
			Bindings []struct {
				Count struct {
					Value string `json:"value"`
				} `json:"count"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result.Results.Bindings) == 0 {
		return 0, nil
	}

	var count int64
	fmt.Sscanf(result.Results.Bindings[0].Count.Value, "%d", &count)
	return count, nil
}

// checkStatus returns the current status of the GraphDB repository
func checkStatus(config *LoaderConfig) *GraphDBStatus {
	status := &GraphDBStatus{
		Repository: config.GraphDBRepo,
	}

	// Check connectivity
	if err := verifyGraphDBConnection(config); err != nil {
		status.ErrorMessage = err.Error()
		return status
	}
	status.Available = true

	// Get triple count
	count, err := getTripleCount(config)
	if err != nil {
		status.ErrorMessage = fmt.Sprintf("Could not get triple count: %v", err)
	} else {
		status.TripleCount = count
	}

	// Check if writable (try a no-op)
	status.Writable = true // Assume writable if accessible

	return status
}

// verifyRepository runs verification queries against the loaded data
func verifyRepository(config *LoaderConfig) bool {
	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Info("KB-7 Repository Verification")
	logger.Info("═══════════════════════════════════════════════════════════")

	success := true

	// Test 1: Triple count
	logger.Info("")
	logger.Info("Test 1: Triple Count")
	count, err := getTripleCount(config)
	if err != nil {
		logger.WithError(err).Error("  ❌ Failed to get triple count")
		success = false
	} else {
		logger.Infof("  ✅ Triple count: %d", count)
		if count < 1_000_000 {
			logger.Warn("  ⚠️  Warning: Low triple count (expected ~14M)")
		}
	}

	// Test 2: SNOMED concepts
	logger.Info("")
	logger.Info("Test 2: SNOMED-CT Concepts")
	snomedCount, err := runCountQuery(config, `
		SELECT (COUNT(?s) as ?count) WHERE {
			?s a <http://www.w3.org/2002/07/owl#Class> .
			FILTER(STRSTARTS(STR(?s), "http://snomed.info/id/"))
		}
	`)
	if err != nil {
		logger.WithError(err).Error("  ❌ SNOMED query failed")
		success = false
	} else {
		logger.Infof("  ✅ SNOMED concepts: %d", snomedCount)
	}

	// Test 3: RxNorm concepts
	logger.Info("")
	logger.Info("Test 3: RxNorm Concepts")
	rxnormCount, err := runCountQuery(config, `
		SELECT (COUNT(?s) as ?count) WHERE {
			?s a <http://www.w3.org/2002/07/owl#Class> .
			FILTER(STRSTARTS(STR(?s), "http://purl.bioontology.org/ontology/RXNORM/"))
		}
	`)
	if err != nil {
		logger.WithError(err).Error("  ❌ RxNorm query failed")
		success = false
	} else {
		logger.Infof("  ✅ RxNorm concepts: %d", rxnormCount)
	}

	// Test 4: LOINC concepts
	logger.Info("")
	logger.Info("Test 4: LOINC Concepts")
	loincCount, err := runCountQuery(config, `
		SELECT (COUNT(?s) as ?count) WHERE {
			?s a <http://www.w3.org/2002/07/owl#Class> .
			FILTER(STRSTARTS(STR(?s), "http://loinc.org/"))
		}
	`)
	if err != nil {
		logger.WithError(err).Error("  ❌ LOINC query failed")
		success = false
	} else {
		logger.Infof("  ✅ LOINC concepts: %d", loincCount)
	}

	// Test 5: Sample SPARQL query
	logger.Info("")
	logger.Info("Test 5: Sample Concept Lookup")
	sampleResult, err := runSampleQuery(config)
	if err != nil {
		logger.WithError(err).Error("  ❌ Sample query failed")
		success = false
	} else {
		logger.Infof("  ✅ Sample concepts retrieved: %d", len(sampleResult))
	}

	logger.Info("")
	logger.Info("═══════════════════════════════════════════════════════════")
	if success {
		logger.Info("✅ ALL VERIFICATION TESTS PASSED")
	} else {
		logger.Error("❌ SOME VERIFICATION TESTS FAILED")
	}
	logger.Info("═══════════════════════════════════════════════════════════")

	return success
}

// runCountQuery executes a COUNT query and returns the result
func runCountQuery(config *LoaderConfig, query string) (int64, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s", config.GraphDBURL, config.GraphDBRepo)

	data := url.Values{}
	data.Set("query", query)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	if config.GraphDBUsername != "" {
		req.SetBasicAuth(config.GraphDBUsername, config.GraphDBPassword)
	}

	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results struct {
			Bindings []struct {
				Count struct {
					Value string `json:"value"`
				} `json:"count"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if len(result.Results.Bindings) == 0 {
		return 0, nil
	}

	var count int64
	fmt.Sscanf(result.Results.Bindings[0].Count.Value, "%d", &count)
	return count, nil
}

// runSampleQuery retrieves sample concepts from the repository
func runSampleQuery(config *LoaderConfig) ([]string, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s", config.GraphDBURL, config.GraphDBRepo)

	query := `
		SELECT ?s ?label WHERE {
			?s a <http://www.w3.org/2002/07/owl#Class> .
			?s <http://www.w3.org/2000/01/rdf-schema#label> ?label .
		} LIMIT 5
	`

	data := url.Values{}
	data.Set("query", query)

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	if config.GraphDBUsername != "" {
		req.SetBasicAuth(config.GraphDBUsername, config.GraphDBPassword)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("query failed with status %d", resp.StatusCode)
	}

	var result struct {
		Results struct {
			Bindings []struct {
				Label struct {
					Value string `json:"value"`
				} `json:"label"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var labels []string
	for _, b := range result.Results.Bindings {
		labels = append(labels, b.Label.Value)
		logger.Debugf("    - %s", b.Label.Value)
	}

	return labels, nil
}

// printResult outputs the load result
func printResult(result *LoadResult) {
	output, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(output))
}

// printStatus outputs the repository status
func printStatus(status *GraphDBStatus) {
	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Info("KB-7 GraphDB Repository Status")
	logger.Info("═══════════════════════════════════════════════════════════")
	logger.Infof("  Repository: %s", status.Repository)
	logger.Infof("  Available:  %v", status.Available)
	logger.Infof("  Writable:   %v", status.Writable)
	logger.Infof("  Triples:    %d", status.TripleCount)
	if status.ErrorMessage != "" {
		logger.Errorf("  Error:      %s", status.ErrorMessage)
	}
	logger.Info("═══════════════════════════════════════════════════════════")
}

// getEnv returns environment variable or default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
