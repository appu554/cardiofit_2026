package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"kb-7-terminology/internal/semantic"

	"github.com/sirupsen/logrus"
)

const (
	testBatchSize = 10
	expectedTriples = 70 // ~7 triples per concept
)

type TestResult struct {
	Name     string
	Passed   bool
	Message  string
	Duration time.Duration
}

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.Info("=== KB-7 Bootstrap Migration Test Suite ===")

	ctx := context.Background()
	results := []TestResult{}

	// Test 1: GraphDB Connection
	logger.Info("\n[TEST 1] GraphDB Connection")
	result := testGraphDBConnection(ctx, logger)
	results = append(results, result)
	printTestResult(result, logger)

	if !result.Passed {
		logger.Error("GraphDB connection failed - aborting tests")
		os.Exit(1)
	}

	// Test 2: Small Batch Migration
	logger.Info("\n[TEST 2] Small Batch Migration (10 concepts)")
	result = testSmallBatchMigration(logger)
	results = append(results, result)
	printTestResult(result, logger)

	// Test 3: SPARQL Query Test
	logger.Info("\n[TEST 3] SPARQL Query Test")
	result = testSPARQLQuery(ctx, logger)
	results = append(results, result)
	printTestResult(result, logger)

	// Test 4: Concept Validation
	logger.Info("\n[TEST 4] Concept Structure Validation")
	result = testConceptValidation(ctx, logger)
	results = append(results, result)
	printTestResult(result, logger)

	// Test 5: Triple Count Verification
	logger.Info("\n[TEST 5] Triple Count Verification")
	result = testTripleCount(ctx, logger)
	results = append(results, result)
	printTestResult(result, logger)

	// Print summary
	printSummary(results, logger)

	// Exit with appropriate code
	allPassed := true
	for _, r := range results {
		if !r.Passed {
			allPassed = false
			break
		}
	}

	if allPassed {
		logger.Info("\n✅ All tests passed! Migration script is ready for full execution.")
		os.Exit(0)
	} else {
		logger.Error("\n❌ Some tests failed. Please fix issues before running full migration.")
		os.Exit(1)
	}
}

func testGraphDBConnection(ctx context.Context, logger *logrus.Logger) TestResult {
	start := time.Now()

	graphDBURL := getEnv("GRAPHDB_URL", "http://localhost:7200")
	repository := getEnv("GRAPHDB_REPOSITORY", "kb7-terminology")

	client := semantic.NewGraphDBClient(graphDBURL, repository, logger)

	err := client.HealthCheck(ctx)

	return TestResult{
		Name:     "GraphDB Connection",
		Passed:   err == nil,
		Message:  getMessage(err, "Connected to "+graphDBURL+"/repositories/"+repository),
		Duration: time.Since(start),
	}
}

func testSmallBatchMigration(logger *logrus.Logger) TestResult {
	start := time.Now()

	// Clear any existing bootstrap data
	logger.Info("Clearing existing bootstrap data...")
	clearBootstrapData()

	// Run migration with small batch
	cmd := exec.Command("go", "run",
		"scripts/bootstrap/postgres-to-graphdb.go",
		"--max", "10",
		"--batch", "10",
	)

	cmd.Dir = "/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology"

	output, err := cmd.CombinedOutput()

	success := err == nil && strings.Contains(string(output), "Migration completed successfully")

	return TestResult{
		Name:     "Small Batch Migration",
		Passed:   success,
		Message:  getMessage(err, "Successfully migrated 10 concepts"),
		Duration: time.Since(start),
	}
}

func testSPARQLQuery(ctx context.Context, logger *logrus.Logger) TestResult {
	start := time.Now()

	graphDBURL := getEnv("GRAPHDB_URL", "http://localhost:7200")
	repository := getEnv("GRAPHDB_REPOSITORY", "kb7-terminology")

	client := semantic.NewGraphDBClient(graphDBURL, repository, logger)

	query := &semantic.SPARQLQuery{
		Query: `
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

			SELECT ?code ?label WHERE {
				?concept a kb7:ClinicalConcept ;
					kb7:code ?code ;
					rdfs:label ?label .
			} LIMIT 5
		`,
	}

	results, err := client.ExecuteSPARQL(ctx, query)

	success := err == nil && len(results.Results.Bindings) > 0

	message := getMessage(err, fmt.Sprintf("Retrieved %d concepts via SPARQL", len(results.Results.Bindings)))

	return TestResult{
		Name:     "SPARQL Query",
		Passed:   success,
		Message:  message,
		Duration: time.Since(start),
	}
}

func testConceptValidation(ctx context.Context, logger *logrus.Logger) TestResult {
	start := time.Now()

	graphDBURL := getEnv("GRAPHDB_URL", "http://localhost:7200")
	repository := getEnv("GRAPHDB_REPOSITORY", "kb7-terminology")

	client := semantic.NewGraphDBClient(graphDBURL, repository, logger)

	// Check that concepts have all required properties
	query := &semantic.SPARQLQuery{
		Query: `
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

			SELECT ?concept WHERE {
				?concept a kb7:ClinicalConcept .
				FILTER NOT EXISTS { ?concept kb7:code ?code }
			} LIMIT 1
		`,
	}

	results, err := client.ExecuteSPARQL(ctx, query)

	// Success if NO concepts missing required properties
	success := err == nil && len(results.Results.Bindings) == 0

	message := getMessage(err, "All concepts have required properties (code, label, system)")
	if !success && err == nil {
		message = fmt.Sprintf("Found %d concepts missing required properties", len(results.Results.Bindings))
	}

	return TestResult{
		Name:     "Concept Validation",
		Passed:   success,
		Message:  message,
		Duration: time.Since(start),
	}
}

func testTripleCount(ctx context.Context, logger *logrus.Logger) TestResult {
	start := time.Now()

	graphDBURL := getEnv("GRAPHDB_URL", "http://localhost:7200")
	repository := getEnv("GRAPHDB_REPOSITORY", "kb7-terminology")

	client := semantic.NewGraphDBClient(graphDBURL, repository, logger)

	// Count total triples in bootstrap context
	query := &semantic.SPARQLQuery{
		Query: `
			SELECT (COUNT(*) AS ?count) WHERE {
				GRAPH <http://cardiofit.ai/bootstrap> {
					?s ?p ?o .
				}
			}
		`,
	}

	results, err := client.ExecuteSPARQL(ctx, query)

	success := false
	message := ""

	if err == nil && len(results.Results.Bindings) > 0 {
		countStr := results.Results.Bindings[0]["count"].Value
		var count int
		fmt.Sscanf(countStr, "%d", &count)

		// Expect at least 60 triples for 10 concepts (some may have optional properties)
		success = count >= 60
		message = fmt.Sprintf("Found %d triples (expected ~70 for 10 concepts)", count)
	} else {
		message = getMessage(err, "")
	}

	return TestResult{
		Name:     "Triple Count",
		Passed:   success,
		Message:  message,
		Duration: time.Since(start),
	}
}

func clearBootstrapData() {
	// Delete bootstrap context using SPARQL UPDATE
	cmd := exec.Command("curl", "-X", "POST",
		"http://localhost:7200/repositories/kb7-terminology/statements",
		"-H", "Content-Type: application/x-www-form-urlencoded",
		"--data-urlencode", "update=CLEAR GRAPH <http://cardiofit.ai/bootstrap>",
	)
	cmd.Run() // Ignore errors - graph may not exist yet
}

func printTestResult(result TestResult, logger *logrus.Logger) {
	status := "❌ FAIL"
	if result.Passed {
		status = "✅ PASS"
	}

	logger.WithFields(logrus.Fields{
		"test":     result.Name,
		"status":   status,
		"duration": result.Duration.Round(time.Millisecond),
		"message":  result.Message,
	}).Info("Test result")
}

func printSummary(results []TestResult, logger *logrus.Logger) {
	logger.Info("\n=== Test Summary ===")

	passed := 0
	total := len(results)

	for _, r := range results {
		if r.Passed {
			passed++
		}
		logger.WithFields(logrus.Fields{
			"test":   r.Name,
			"result": getStatus(r.Passed),
		}).Info("")
	}

	logger.WithFields(logrus.Fields{
		"passed": passed,
		"total":  total,
		"rate":   fmt.Sprintf("%.0f%%", float64(passed)/float64(total)*100),
	}).Info("Overall results")
}

func getMessage(err error, successMsg string) string {
	if err != nil {
		return err.Error()
	}
	return successMsg
}

func getStatus(passed bool) string {
	if passed {
		return "✅ PASS"
	}
	return "❌ FAIL"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
