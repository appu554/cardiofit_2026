package bulkload

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// DataIntegrityValidator performs comprehensive data integrity checks
type DataIntegrityValidator struct {
	postgresDB    *sql.DB
	elasticsearch *elasticsearch.Client
	logger        *logrus.Logger
	config        *IntegrityConfig
	results       *IntegrityResults
	mu            sync.RWMutex
}

// IntegrityConfig contains validation configuration
type IntegrityConfig struct {
	IndexName           string   `json:"index_name"`
	SamplePercentage    float64  `json:"sample_percentage"`
	MaxSampleSize       int      `json:"max_sample_size"`
	ParallelValidations int      `json:"parallel_validations"`
	StrictMode          bool     `json:"strict_mode"`
	Systems             []string `json:"systems"`
	ValidationTypes     []string `json:"validation_types"`
	FailureThreshold    float64  `json:"failure_threshold"`
	DetailedReporting   bool     `json:"detailed_reporting"`
}

// IntegrityResults contains validation results
type IntegrityResults struct {
	StartTime         time.Time                `json:"start_time"`
	EndTime           time.Time                `json:"end_time"`
	TotalChecks       int64                    `json:"total_checks"`
	PassedChecks      int64                    `json:"passed_checks"`
	FailedChecks      int64                    `json:"failed_checks"`
	ValidationDetails []ValidationDetail       `json:"validation_details"`
	SystemResults     map[string]SystemResult  `json:"system_results"`
	Discrepancies     []DataDiscrepancy        `json:"discrepancies"`
	Summary           ValidationSummary        `json:"summary"`
}

// ValidationDetail contains details of a single validation
type ValidationDetail struct {
	Timestamp      time.Time `json:"timestamp"`
	ValidationType string    `json:"validation_type"`
	System         string    `json:"system,omitempty"`
	Passed         bool      `json:"passed"`
	Message        string    `json:"message"`
	SampleSize     int       `json:"sample_size,omitempty"`
	ErrorCount     int       `json:"error_count,omitempty"`
	Duration       time.Duration `json:"duration"`
}

// SystemResult contains validation results for a specific system
type SystemResult struct {
	System              string    `json:"system"`
	PostgresCount       int64     `json:"postgres_count"`
	ElasticsearchCount  int64     `json:"elasticsearch_count"`
	MatchedRecords      int64     `json:"matched_records"`
	MismatchedRecords   int64     `json:"mismatched_records"`
	MissingInES         int64     `json:"missing_in_elasticsearch"`
	ExtraInES           int64     `json:"extra_in_elasticsearch"`
	LastValidated       time.Time `json:"last_validated"`
}

// DataDiscrepancy represents a data mismatch
type DataDiscrepancy struct {
	System         string                 `json:"system"`
	Code           string                 `json:"code"`
	Field          string                 `json:"field"`
	PostgresValue  interface{}            `json:"postgres_value"`
	ElasticsearchValue interface{}        `json:"elasticsearch_value"`
	Severity       string                 `json:"severity"` // "critical", "major", "minor"
	DetectedAt     time.Time              `json:"detected_at"`
}

// ValidationSummary provides high-level validation summary
type ValidationSummary struct {
	OverallStatus       string    `json:"overall_status"` // "passed", "failed", "warning"
	DataConsistency     float64   `json:"data_consistency_percentage"`
	RecommendedActions  []string  `json:"recommended_actions"`
	EstimatedImpact     string    `json:"estimated_impact"`
	CanProceed          bool      `json:"can_proceed"`
}

// ValidationType defines types of validation checks
type ValidationType string

const (
	ValidationTypeCount      ValidationType = "record_count"
	ValidationTypeChecksum   ValidationType = "checksum"
	ValidationTypeSample     ValidationType = "sample_comparison"
	ValidationTypeSearch     ValidationType = "search_functionality"
	ValidationTypeRelations  ValidationType = "relationships"
	ValidationTypePerformance ValidationType = "performance"
)

// NewDataIntegrityValidator creates a new validator
func NewDataIntegrityValidator(
	postgresDB *sql.DB,
	elasticsearch *elasticsearch.Client,
	logger *logrus.Logger,
	config *IntegrityConfig,
) *DataIntegrityValidator {
	return &DataIntegrityValidator{
		postgresDB:    postgresDB,
		elasticsearch: elasticsearch,
		logger:        logger,
		config:        config,
		results: &IntegrityResults{
			StartTime:         time.Now(),
			ValidationDetails: make([]ValidationDetail, 0),
			SystemResults:     make(map[string]SystemResult),
			Discrepancies:     make([]DataDiscrepancy, 0),
		},
	}
}

// ValidateAll performs all configured validation checks
func (v *DataIntegrityValidator) ValidateAll(ctx context.Context) (*IntegrityResults, error) {
	v.logger.Info("Starting comprehensive data integrity validation")

	// Determine validation types
	validationTypes := v.config.ValidationTypes
	if len(validationTypes) == 0 {
		validationTypes = []string{
			string(ValidationTypeCount),
			string(ValidationTypeChecksum),
			string(ValidationTypeSample),
			string(ValidationTypeSearch),
		}
	}

	// Execute validations
	for _, validationType := range validationTypes {
		switch ValidationType(validationType) {
		case ValidationTypeCount:
			v.validateRecordCounts(ctx)
		case ValidationTypeChecksum:
			v.validateChecksums(ctx)
		case ValidationTypeSample:
			v.validateSampleRecords(ctx)
		case ValidationTypeSearch:
			v.validateSearchFunctionality(ctx)
		case ValidationTypeRelations:
			v.validateRelationships(ctx)
		case ValidationTypePerformance:
			v.validatePerformance(ctx)
		default:
			v.logger.Warnf("Unknown validation type: %s", validationType)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return v.results, ctx.Err()
		default:
		}
	}

	// Generate summary
	v.generateSummary()

	v.results.EndTime = time.Now()
	return v.results, nil
}

// validateRecordCounts compares record counts between stores
func (v *DataIntegrityValidator) validateRecordCounts(ctx context.Context) {
	start := time.Now()
	v.logger.Info("Validating record counts...")

	systems := v.config.Systems
	if len(systems) == 0 {
		systems = []string{"SNOMED", "RxNorm", "LOINC", "ICD10", "ICD9", "CPT", "NDC"}
	}

	allPassed := true
	for _, system := range systems {
		// Get PostgreSQL count
		var pgCount int64
		query := "SELECT COUNT(*) FROM concepts WHERE system = $1"
		err := v.postgresDB.QueryRowContext(ctx, query, system).Scan(&pgCount)
		if err != nil {
			v.logger.WithError(err).Errorf("Failed to get PostgreSQL count for %s", system)
			continue
		}

		// Get Elasticsearch count
		esCount, err := v.getElasticsearchCount(ctx, system)
		if err != nil {
			v.logger.WithError(err).Errorf("Failed to get Elasticsearch count for %s", system)
			continue
		}

		// Compare counts
		diff := pgCount - esCount
		passed := diff == 0

		if !v.config.StrictMode && diff != 0 {
			// Allow small differences in non-strict mode
			errorRate := float64(abs(diff)) / float64(pgCount)
			passed = errorRate < v.config.FailureThreshold
		}

		if !passed {
			allPassed = false
			v.recordDiscrepancy(DataDiscrepancy{
				System:             system,
				Field:              "record_count",
				PostgresValue:      pgCount,
				ElasticsearchValue: esCount,
				Severity:           "major",
				DetectedAt:         time.Now(),
			})
		}

		// Update system results
		v.mu.Lock()
		v.results.SystemResults[system] = SystemResult{
			System:             system,
			PostgresCount:      pgCount,
			ElasticsearchCount: esCount,
			LastValidated:      time.Now(),
		}
		v.mu.Unlock()

		v.logger.Infof("System %s: PostgreSQL=%d, Elasticsearch=%d, Diff=%d",
			system, pgCount, esCount, diff)
	}

	v.recordValidation(ValidationDetail{
		Timestamp:      time.Now(),
		ValidationType: string(ValidationTypeCount),
		Passed:         allPassed,
		Message:        "Record count validation completed",
		Duration:       time.Since(start),
	})
}

// validateChecksums validates data integrity using checksums
func (v *DataIntegrityValidator) validateChecksums(ctx context.Context) {
	start := time.Now()
	v.logger.Info("Validating data checksums...")

	// Sample records for checksum validation
	sampleSize := v.calculateSampleSize(ctx)

	// Get random sample from PostgreSQL
	pgSample, err := v.getPostgresSample(ctx, sampleSize)
	if err != nil {
		v.logger.WithError(err).Error("Failed to get PostgreSQL sample")
		v.recordValidation(ValidationDetail{
			Timestamp:      time.Now(),
			ValidationType: string(ValidationTypeChecksum),
			Passed:         false,
			Message:        "Failed to get PostgreSQL sample",
			Duration:       time.Since(start),
		})
		return
	}

	// Validate each record's checksum
	errorCount := 0
	for _, pgRecord := range pgSample {
		esRecord, err := v.getElasticsearchRecord(ctx, pgRecord.System, pgRecord.Code)
		if err != nil {
			errorCount++
			continue
		}

		pgChecksum := v.calculateChecksum(pgRecord)
		esChecksum := v.calculateChecksumFromES(esRecord)

		if pgChecksum != esChecksum {
			errorCount++
			v.recordDiscrepancy(DataDiscrepancy{
				System:             pgRecord.System,
				Code:              pgRecord.Code,
				Field:             "checksum",
				PostgresValue:      pgChecksum,
				ElasticsearchValue: esChecksum,
				Severity:          "critical",
				DetectedAt:        time.Now(),
			})
		}
	}

	passed := errorCount == 0
	if !v.config.StrictMode && errorCount > 0 {
		errorRate := float64(errorCount) / float64(sampleSize)
		passed = errorRate < v.config.FailureThreshold
	}

	v.recordValidation(ValidationDetail{
		Timestamp:      time.Now(),
		ValidationType: string(ValidationTypeChecksum),
		Passed:         passed,
		Message:        fmt.Sprintf("Checksum validation: %d errors in %d samples", errorCount, sampleSize),
		SampleSize:     sampleSize,
		ErrorCount:     errorCount,
		Duration:       time.Since(start),
	})
}

// validateSampleRecords performs detailed field-by-field validation
func (v *DataIntegrityValidator) validateSampleRecords(ctx context.Context) {
	start := time.Now()
	v.logger.Info("Validating sample records...")

	sampleSize := v.calculateSampleSize(ctx)
	pgSample, err := v.getPostgresSample(ctx, sampleSize)
	if err != nil {
		v.logger.WithError(err).Error("Failed to get PostgreSQL sample")
		return
	}

	errorCount := 0
	fieldsToCheck := []string{"display", "status", "synonyms", "definition", "domain"}

	for _, pgRecord := range pgSample {
		esRecord, err := v.getElasticsearchRecord(ctx, pgRecord.System, pgRecord.Code)
		if err != nil {
			errorCount++
			v.recordDiscrepancy(DataDiscrepancy{
				System:         pgRecord.System,
				Code:          pgRecord.Code,
				Field:         "existence",
				PostgresValue:  "exists",
				ElasticsearchValue: "missing",
				Severity:      "critical",
				DetectedAt:    time.Now(),
			})
			continue
		}

		// Compare fields
		for _, field := range fieldsToCheck {
			pgValue := v.getFieldValue(pgRecord, field)
			esValue := v.getFieldValueFromES(esRecord, field)

			if !v.compareValues(pgValue, esValue) {
				errorCount++
				v.recordDiscrepancy(DataDiscrepancy{
					System:         pgRecord.System,
					Code:          pgRecord.Code,
					Field:         field,
					PostgresValue:  pgValue,
					ElasticsearchValue: esValue,
					Severity:      "major",
					DetectedAt:    time.Now(),
				})
			}
		}
	}

	passed := errorCount == 0
	if !v.config.StrictMode && errorCount > 0 {
		errorRate := float64(errorCount) / float64(sampleSize * len(fieldsToCheck))
		passed = errorRate < v.config.FailureThreshold
	}

	v.recordValidation(ValidationDetail{
		Timestamp:      time.Now(),
		ValidationType: string(ValidationTypeSample),
		Passed:         passed,
		Message:        fmt.Sprintf("Sample validation: %d field mismatches in %d samples", errorCount, sampleSize),
		SampleSize:     sampleSize,
		ErrorCount:     errorCount,
		Duration:       time.Since(start),
	})
}

// validateSearchFunctionality tests search capabilities
func (v *DataIntegrityValidator) validateSearchFunctionality(ctx context.Context) {
	start := time.Now()
	v.logger.Info("Validating search functionality...")

	testQueries := []struct {
		Query       string
		System      string
		ExpectedMin int
	}{
		{"hypertension", "SNOMED", 1},
		{"diabetes", "ICD10", 1},
		{"aspirin", "RxNorm", 1},
		{"glucose", "LOINC", 1},
	}

	failedTests := 0
	for _, test := range testQueries {
		// Search in Elasticsearch
		results, err := v.searchElasticsearch(ctx, test.Query, test.System)
		if err != nil {
			v.logger.WithError(err).Errorf("Search test failed for query: %s", test.Query)
			failedTests++
			continue
		}

		if len(results) < test.ExpectedMin {
			failedTests++
			v.logger.Warnf("Search test failed: query='%s', system='%s', expected>=%d, got=%d",
				test.Query, test.System, test.ExpectedMin, len(results))
		}
	}

	passed := failedTests == 0

	v.recordValidation(ValidationDetail{
		Timestamp:      time.Now(),
		ValidationType: string(ValidationTypeSearch),
		Passed:         passed,
		Message:        fmt.Sprintf("Search validation: %d/%d tests passed", len(testQueries)-failedTests, len(testQueries)),
		ErrorCount:     failedTests,
		Duration:       time.Since(start),
	})
}

// validateRelationships validates hierarchical relationships
func (v *DataIntegrityValidator) validateRelationships(ctx context.Context) {
	start := time.Now()
	v.logger.Info("Validating hierarchical relationships...")

	// Sample concepts with parent relationships
	query := `
		SELECT system, code, parent_codes
		FROM concepts
		WHERE array_length(parent_codes, 1) > 0
		ORDER BY RANDOM()
		LIMIT 100
	`

	rows, err := v.postgresDB.QueryContext(ctx, query)
	if err != nil {
		v.logger.WithError(err).Error("Failed to query relationships")
		return
	}
	defer rows.Close()

	errorCount := 0
	checkCount := 0

	for rows.Next() {
		var system, code string
		var parentCodes pq.StringArray

		if err := rows.Scan(&system, &code, &parentCodes); err != nil {
			continue
		}

		checkCount++

		// Verify relationships in Elasticsearch
		esRecord, err := v.getElasticsearchRecord(ctx, system, code)
		if err != nil {
			errorCount++
			continue
		}

		esParents := v.getParentCodesFromES(esRecord)
		if !v.compareStringArrays([]string(parentCodes), esParents) {
			errorCount++
			v.recordDiscrepancy(DataDiscrepancy{
				System:             system,
				Code:              code,
				Field:             "parent_codes",
				PostgresValue:      parentCodes,
				ElasticsearchValue: esParents,
				Severity:          "minor",
				DetectedAt:        time.Now(),
			})
		}
	}

	passed := errorCount == 0
	if !v.config.StrictMode && errorCount > 0 {
		errorRate := float64(errorCount) / float64(checkCount)
		passed = errorRate < v.config.FailureThreshold
	}

	v.recordValidation(ValidationDetail{
		Timestamp:      time.Now(),
		ValidationType: string(ValidationTypeRelations),
		Passed:         passed,
		Message:        fmt.Sprintf("Relationship validation: %d errors in %d checks", errorCount, checkCount),
		SampleSize:     checkCount,
		ErrorCount:     errorCount,
		Duration:       time.Since(start),
	})
}

// validatePerformance tests query performance
func (v *DataIntegrityValidator) validatePerformance(ctx context.Context) {
	start := time.Now()
	v.logger.Info("Validating query performance...")

	// Test various query types
	performanceTests := []struct {
		Name          string
		TestFunc      func(context.Context) (time.Duration, error)
		MaxDuration   time.Duration
	}{
		{
			Name: "Exact Code Lookup",
			TestFunc: func(ctx context.Context) (time.Duration, error) {
				start := time.Now()
				_, err := v.getElasticsearchRecord(ctx, "SNOMED", "387517004")
				return time.Since(start), err
			},
			MaxDuration: 50 * time.Millisecond,
		},
		{
			Name: "Text Search",
			TestFunc: func(ctx context.Context) (time.Duration, error) {
				start := time.Now()
				_, err := v.searchElasticsearch(ctx, "hypertension", "")
				return time.Since(start), err
			},
			MaxDuration: 200 * time.Millisecond,
		},
	}

	failedTests := 0
	for _, test := range performanceTests {
		duration, err := test.TestFunc(ctx)
		if err != nil {
			failedTests++
			v.logger.WithError(err).Errorf("Performance test failed: %s", test.Name)
			continue
		}

		if duration > test.MaxDuration {
			failedTests++
			v.logger.Warnf("Performance test slow: %s took %v (max: %v)",
				test.Name, duration, test.MaxDuration)
		} else {
			v.logger.Infof("Performance test passed: %s took %v", test.Name, duration)
		}
	}

	passed := failedTests == 0

	v.recordValidation(ValidationDetail{
		Timestamp:      time.Now(),
		ValidationType: string(ValidationTypePerformance),
		Passed:         passed,
		Message:        fmt.Sprintf("Performance validation: %d/%d tests passed", len(performanceTests)-failedTests, len(performanceTests)),
		ErrorCount:     failedTests,
		Duration:       time.Since(start),
	})
}

// Helper methods

func (v *DataIntegrityValidator) calculateSampleSize(ctx context.Context) int {
	// Get total record count
	var totalCount int64
	query := "SELECT COUNT(*) FROM concepts"
	v.postgresDB.QueryRowContext(ctx, query).Scan(&totalCount)

	// Calculate sample size based on percentage
	sampleSize := int(float64(totalCount) * v.config.SamplePercentage)

	// Apply min/max constraints
	if sampleSize < 100 {
		sampleSize = 100
	}
	if sampleSize > v.config.MaxSampleSize {
		sampleSize = v.config.MaxSampleSize
	}

	return sampleSize
}

func (v *DataIntegrityValidator) getPostgresSample(ctx context.Context, size int) ([]*ConceptRecord, error) {
	query := `
		SELECT id, concept_uuid, system, code, version, preferred_term,
		       synonyms, COALESCE(properties->>'definition', '') as definition,
		       parent_codes, active, properties,
		       COALESCE(properties->>'domain', '') as domain
		FROM concepts
		ORDER BY RANDOM()
		LIMIT $1
	`

	rows, err := v.postgresDB.QueryContext(ctx, query, size)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*ConceptRecord, 0, size)
	for rows.Next() {
		record := &ConceptRecord{}
		// Scan record fields...
		// (Implementation similar to bulk_loader.go)
		records = append(records, record)
	}

	return records, rows.Err()
}

func (v *DataIntegrityValidator) getElasticsearchCount(ctx context.Context, system string) (int64, error) {
	// Build query
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"system": system,
			},
		},
	}

	queryJSON, _ := json.Marshal(query)

	// Execute count query
	res, err := v.elasticsearch.Count(
		v.elasticsearch.Count.WithContext(ctx),
		v.elasticsearch.Count.WithIndex(v.config.IndexName),
		v.elasticsearch.Count.WithBody(strings.NewReader(string(queryJSON))),
	)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return 0, err
	}

	count, ok := result["count"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid count response")
	}

	return int64(count), nil
}

func (v *DataIntegrityValidator) getElasticsearchRecord(ctx context.Context, system, code string) (map[string]interface{}, error) {
	// Get document by ID
	docID := fmt.Sprintf("%s_%s", system, code)

	res, err := v.elasticsearch.Get(
		v.config.IndexName,
		docID,
		v.elasticsearch.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, fmt.Errorf("record not found")
	}

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	source, ok := response["_source"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	return source, nil
}

func (v *DataIntegrityValidator) searchElasticsearch(ctx context.Context, query, system string) ([]map[string]interface{}, error) {
	// Build search query
	searchQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"display": query,
						},
					},
				},
			},
		},
		"size": 10,
	}

	if system != "" {
		searchQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = map[string]interface{}{
			"term": map[string]interface{}{
				"system": system,
			},
		}
	}

	queryJSON, _ := json.Marshal(searchQuery)

	// Execute search
	res, err := v.elasticsearch.Search(
		v.elasticsearch.Search.WithContext(ctx),
		v.elasticsearch.Search.WithIndex(v.config.IndexName),
		v.elasticsearch.Search.WithBody(strings.NewReader(string(queryJSON))),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var response map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}

	hits, _ := response["hits"].(map[string]interface{})["hits"].([]interface{})
	results := make([]map[string]interface{}, 0, len(hits))

	for _, hit := range hits {
		if source, ok := hit.(map[string]interface{})["_source"].(map[string]interface{}); ok {
			results = append(results, source)
		}
	}

	return results, nil
}

func (v *DataIntegrityValidator) calculateChecksum(record *ConceptRecord) string {
	// Create consistent string representation
	data := fmt.Sprintf("%s|%s|%s|%s|%v",
		record.System,
		record.Code,
		record.PreferredTerm,
		strings.Join(record.Synonyms, ","),
		record.Active,
	)

	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (v *DataIntegrityValidator) calculateChecksumFromES(record map[string]interface{}) string {
	// Extract fields and create checksum
	system, _ := record["system"].(string)
	code, _ := record["code"].(string)
	display, _ := record["display"].(string)

	synonyms := []string{}
	if synArray, ok := record["synonyms"].([]interface{}); ok {
		for _, syn := range synArray {
			if s, ok := syn.(string); ok {
				synonyms = append(synonyms, s)
			}
		}
	}

	status, _ := record["status"].(string)
	active := status == "active"

	data := fmt.Sprintf("%s|%s|%s|%s|%v",
		system,
		code,
		display,
		strings.Join(synonyms, ","),
		active,
	)

	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (v *DataIntegrityValidator) getFieldValue(record *ConceptRecord, field string) interface{} {
	switch field {
	case "display":
		return record.PreferredTerm
	case "status":
		if record.Active {
			return "active"
		}
		return "inactive"
	case "synonyms":
		return record.Synonyms
	case "definition":
		return record.Definition
	case "domain":
		return record.Domain
	default:
		return nil
	}
}

func (v *DataIntegrityValidator) getFieldValueFromES(record map[string]interface{}, field string) interface{} {
	return record[field]
}

func (v *DataIntegrityValidator) compareValues(v1, v2 interface{}) bool {
	// Handle different types of comparisons
	switch val1 := v1.(type) {
	case string:
		val2, ok := v2.(string)
		return ok && val1 == val2
	case []string:
		val2, ok := v2.([]interface{})
		if !ok {
			return false
		}
		return v.compareStringArrays(val1, interfaceArrayToStringArray(val2))
	default:
		return fmt.Sprintf("%v", v1) == fmt.Sprintf("%v", v2)
	}
}

func (v *DataIntegrityValidator) compareStringArrays(a1, a2 []string) bool {
	if len(a1) != len(a2) {
		return false
	}

	// Create maps for comparison
	m1 := make(map[string]bool)
	for _, s := range a1 {
		m1[s] = true
	}

	for _, s := range a2 {
		if !m1[s] {
			return false
		}
	}

	return true
}

func (v *DataIntegrityValidator) getParentCodesFromES(record map[string]interface{}) []string {
	parentCodes := []string{}
	if parents, ok := record["parent_codes"].([]interface{}); ok {
		for _, p := range parents {
			if code, ok := p.(string); ok {
				parentCodes = append(parentCodes, code)
			}
		}
	}
	return parentCodes
}

func (v *DataIntegrityValidator) recordValidation(detail ValidationDetail) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.results.ValidationDetails = append(v.results.ValidationDetails, detail)
	v.results.TotalChecks++
	if detail.Passed {
		v.results.PassedChecks++
	} else {
		v.results.FailedChecks++
	}
}

func (v *DataIntegrityValidator) recordDiscrepancy(discrepancy DataDiscrepancy) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.results.Discrepancies = append(v.results.Discrepancies, discrepancy)

	// Keep only last 1000 discrepancies to prevent memory issues
	if len(v.results.Discrepancies) > 1000 {
		v.results.Discrepancies = v.results.Discrepancies[len(v.results.Discrepancies)-1000:]
	}
}

func (v *DataIntegrityValidator) generateSummary() {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Calculate consistency percentage
	consistency := float64(v.results.PassedChecks) / float64(v.results.TotalChecks) * 100

	// Determine overall status
	status := "passed"
	if consistency < 95 {
		status = "warning"
	}
	if consistency < 90 {
		status = "failed"
	}

	// Generate recommendations
	recommendations := []string{}
	if len(v.results.Discrepancies) > 0 {
		recommendations = append(recommendations, "Review and fix data discrepancies")
	}
	if consistency < 100 {
		recommendations = append(recommendations, "Re-run bulk load for affected systems")
	}

	// Estimate impact
	impact := "minimal"
	if consistency < 95 {
		impact = "moderate"
	}
	if consistency < 90 {
		impact = "severe"
	}

	v.results.Summary = ValidationSummary{
		OverallStatus:      status,
		DataConsistency:    consistency,
		RecommendedActions: recommendations,
		EstimatedImpact:    impact,
		CanProceed:        consistency >= 90,
	}
}

// Helper functions

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func interfaceArrayToStringArray(arr []interface{}) []string {
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}