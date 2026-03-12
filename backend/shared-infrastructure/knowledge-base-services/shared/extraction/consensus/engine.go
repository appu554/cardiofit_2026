// Package consensus provides the Race-to-Consensus Engine for multi-LLM extraction.
//
// Phase 3c.3: Race-to-Consensus Engine
// Authority Level: CONSENSUS REQUIRED for all LLM-extracted facts
//
// KEY PRINCIPLE: "LLMs disagree → HUMAN first" (Navigation Rule 3)
// No single LLM's extraction is accepted without corroboration.
//
// CONSENSUS REQUIREMENTS:
// - Minimum 2 of 3 providers must agree
// - Agreement is based on semantic equivalence, not exact match
// - Numeric values allow configurable tolerance (default 5%)
// - Disagreements are flagged for human review
package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/cardiofit/shared/extraction/llm"
)

// =============================================================================
// CONSENSUS ENGINE
// =============================================================================

// Engine coordinates multi-LLM extraction with consensus requirements
type Engine struct {
	providers       []llm.Provider
	minAgreement    int           // Minimum providers that must agree (default: 2)
	timeout         time.Duration // Timeout for the entire consensus operation
	numericTolerance float64      // Max difference in numeric values to consider "agreement" (default: 0.05 = 5%)
	stringNormalize  bool         // Normalize strings before comparison (case, whitespace)
}

// Config contains consensus engine configuration
type Config struct {
	// MinAgreement is the minimum number of providers that must agree
	// Default: 2 (for 2-of-3 consensus)
	MinAgreement int

	// Timeout is the maximum time for consensus extraction
	// Default: 60 seconds
	Timeout time.Duration

	// NumericTolerance is the maximum relative difference for numeric agreement
	// Default: 0.05 (5%)
	NumericTolerance float64

	// StringNormalize enables case-insensitive, whitespace-normalized string comparison
	// Default: true
	StringNormalize bool
}

// DefaultConfig returns default consensus configuration
func DefaultConfig() Config {
	return Config{
		MinAgreement:     2,
		Timeout:          60 * time.Second,
		NumericTolerance: 0.05,
		StringNormalize:  true,
	}
}

// NewEngine creates a consensus engine with the specified providers
func NewEngine(providers []llm.Provider, config ...Config) *Engine {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &Engine{
		providers:        providers,
		minAgreement:     cfg.MinAgreement,
		timeout:          cfg.Timeout,
		numericTolerance: cfg.NumericTolerance,
		stringNormalize:  cfg.StringNormalize,
	}
}

// =============================================================================
// CONSENSUS RESULT
// =============================================================================

// Result contains the consensus extraction outcome
type Result struct {
	// ─────────────────────────────────────────────────────────────────────────
	// CONSENSUS STATUS
	// ─────────────────────────────────────────────────────────────────────────

	// Achieved is true if consensus was reached
	Achieved bool `json:"achieved"`

	// AgreementCount is the number of providers that agreed
	AgreementCount int `json:"agreementCount"`

	// TotalProviders is the total number of providers queried
	TotalProviders int `json:"totalProviders"`

	// ─────────────────────────────────────────────────────────────────────────
	// WINNING VALUE
	// ─────────────────────────────────────────────────────────────────────────

	// WinningValue is the agreed-upon extraction (if consensus achieved)
	WinningValue interface{} `json:"winningValue,omitempty"`

	// WinningValueJSON is the JSON representation of the winning value
	WinningValueJSON json.RawMessage `json:"winningValueJson,omitempty"`

	// Confidence is the confidence of the consensus result
	Confidence float64 `json:"confidence"`

	// MaxConfidence is the highest confidence among agreeing providers
	MaxConfidence float64 `json:"maxConfidence"`

	// ─────────────────────────────────────────────────────────────────────────
	// DISAGREEMENTS
	// ─────────────────────────────────────────────────────────────────────────

	// Disagreements lists fields where providers disagreed
	Disagreements []Disagreement `json:"disagreements,omitempty"`

	// RequiresHuman is true if human review is needed
	RequiresHuman bool `json:"requiresHuman"`

	// ─────────────────────────────────────────────────────────────────────────
	// PROVIDER RESULTS
	// ─────────────────────────────────────────────────────────────────────────

	// ProviderResults contains the individual results from each provider
	ProviderResults []llm.ExtractionResult `json:"providerResults"`

	// SuccessfulProviders is the count of providers that returned results
	SuccessfulProviders int `json:"successfulProviders"`

	// FailedProviders contains error messages from failed providers
	FailedProviders map[string]string `json:"failedProviders,omitempty"`

	// ─────────────────────────────────────────────────────────────────────────
	// METADATA
	// ─────────────────────────────────────────────────────────────────────────

	// TotalLatency is the time taken for the consensus operation
	TotalLatency time.Duration `json:"totalLatency"`

	// TotalTokens is the sum of tokens used by all providers
	TotalTokens int `json:"totalTokens"`

	// TotalCost is the estimated total cost in USD
	TotalCost float64 `json:"totalCost"`

	// CompletedAt is when consensus was determined
	CompletedAt time.Time `json:"completedAt"`
}

// Disagreement represents a field where providers disagreed
type Disagreement struct {
	// Field is the JSON path to the disagreeing field
	Field string `json:"field"`

	// Provider1 is the first provider's name
	Provider1 string `json:"provider1"`

	// Value1 is the first provider's value
	Value1 interface{} `json:"value1"`

	// Provider2 is the second provider's name
	Provider2 string `json:"provider2"`

	// Value2 is the second provider's value
	Value2 interface{} `json:"value2"`

	// Severity indicates how critical the disagreement is
	// CRITICAL: Values are significantly different (e.g., 50% vs 100% dose)
	// MINOR: Values are similar but not within tolerance (e.g., 5.1 vs 5.3)
	Severity DisagreementSeverity `json:"severity"`

	// ClinicalImpact describes the potential clinical significance
	ClinicalImpact string `json:"clinicalImpact,omitempty"`
}

// DisagreementSeverity categorizes disagreement severity
type DisagreementSeverity string

const (
	// SeverityCritical indicates a significant clinical difference
	SeverityCritical DisagreementSeverity = "CRITICAL"

	// SeverityMinor indicates a minor difference
	SeverityMinor DisagreementSeverity = "MINOR"
)

// =============================================================================
// EXTRACTION METHODS
// =============================================================================

// Extract runs parallel extraction across all providers and checks consensus
func (e *Engine) Extract(ctx context.Context, req *llm.ExtractionRequest) (*Result, error) {
	startTime := time.Now()

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	result := &Result{
		TotalProviders:  len(e.providers),
		ProviderResults: make([]llm.ExtractionResult, 0, len(e.providers)),
		FailedProviders: make(map[string]string),
	}

	// Run providers in parallel
	results := make([]llm.ExtractionResult, len(e.providers))
	errors := make([]error, len(e.providers))
	var wg sync.WaitGroup

	for i, provider := range e.providers {
		wg.Add(1)
		go func(idx int, p llm.Provider) {
			defer wg.Done()
			res, err := p.Extract(ctx, req)
			if err != nil {
				errors[idx] = err
				return
			}
			results[idx] = *res
		}(i, provider)
	}

	wg.Wait()

	// Collect successful results
	successResults := make([]llm.ExtractionResult, 0, len(e.providers))
	for i, err := range errors {
		if err != nil {
			result.FailedProviders[e.providers[i].Name()] = err.Error()
		} else if results[i].Success {
			successResults = append(successResults, results[i])
			result.ProviderResults = append(result.ProviderResults, results[i])
		} else {
			result.FailedProviders[e.providers[i].Name()] = results[i].Error
		}
	}

	result.SuccessfulProviders = len(successResults)

	// Check if we have enough responses
	if len(successResults) < e.minAgreement {
		result.Achieved = false
		result.RequiresHuman = true
		result.TotalLatency = time.Since(startTime)
		result.CompletedAt = time.Now()
		return result, fmt.Errorf("insufficient provider responses: %d/%d required",
			len(successResults), e.minAgreement)
	}

	// Check consensus
	consensusResult := e.checkConsensus(successResults)
	result.Achieved = consensusResult.Achieved
	result.AgreementCount = consensusResult.AgreementCount
	result.WinningValue = consensusResult.WinningValue
	result.WinningValueJSON = consensusResult.WinningValueJSON
	result.Confidence = consensusResult.Confidence
	result.MaxConfidence = consensusResult.MaxConfidence
	result.Disagreements = consensusResult.Disagreements
	result.RequiresHuman = !consensusResult.Achieved

	// Calculate totals
	for _, r := range result.ProviderResults {
		result.TotalTokens += r.TokensUsed.TotalTokens
		result.TotalCost += r.Cost
	}

	result.TotalLatency = time.Since(startTime)
	result.CompletedAt = time.Now()

	return result, nil
}

// =============================================================================
// CONSENSUS CHECKING
// =============================================================================

func (e *Engine) checkConsensus(results []llm.ExtractionResult) *Result {
	result := &Result{
		Disagreements: make([]Disagreement, 0),
	}

	if len(results) < 2 {
		result.Achieved = false
		return result
	}

	// Group results by semantic equivalence
	groups := e.groupByAgreement(results)

	// Find largest agreement group
	var largestGroup []llm.ExtractionResult
	for _, group := range groups {
		if len(group) > len(largestGroup) {
			largestGroup = group
		}
	}

	result.AgreementCount = len(largestGroup)
	result.Achieved = len(largestGroup) >= e.minAgreement

	if result.Achieved {
		// Use highest confidence result from agreeing providers
		var maxConf float64
		for _, r := range largestGroup {
			if r.Confidence > maxConf {
				maxConf = r.Confidence
				result.WinningValue = r.ExtractedData
				result.WinningValueJSON = r.ExtractedDataJSON
				result.Confidence = r.Confidence
			}
		}
		result.MaxConfidence = maxConf
	} else {
		// Find and record disagreements
		result.Disagreements = e.findDisagreements(results)
	}

	return result
}

// groupByAgreement groups results by semantic equivalence
func (e *Engine) groupByAgreement(results []llm.ExtractionResult) [][]llm.ExtractionResult {
	if len(results) == 0 {
		return nil
	}

	// Use a simple grouping algorithm: check if each result agrees with existing groups
	groups := make([][]llm.ExtractionResult, 0)

	for _, result := range results {
		foundGroup := false
		for i, group := range groups {
			if len(group) > 0 && e.semanticallyEqual(result.ExtractedData, group[0].ExtractedData) {
				groups[i] = append(groups[i], result)
				foundGroup = true
				break
			}
		}
		if !foundGroup {
			groups = append(groups, []llm.ExtractionResult{result})
		}
	}

	return groups
}

// semanticallyEqual checks if two extracted values are semantically equivalent
func (e *Engine) semanticallyEqual(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Convert to JSON for comparison
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}

	// Parse as generic maps for deep comparison
	var aMap, bMap interface{}
	if err := json.Unmarshal(aJSON, &aMap); err != nil {
		return false
	}
	if err := json.Unmarshal(bJSON, &bMap); err != nil {
		return false
	}

	return e.deepEqual(aMap, bMap, "")
}

// deepEqual recursively compares two values with tolerance for numerics
func (e *Engine) deepEqual(a, b interface{}, path string) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Check types match
	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)

	if aType != bType {
		// Special case: both numeric but different types (float64 vs int)
		aNum, aIsNum := toFloat64(a)
		bNum, bIsNum := toFloat64(b)
		if aIsNum && bIsNum {
			return e.numericEqual(aNum, bNum)
		}
		return false
	}

	// Compare based on type
	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal := b.(map[string]interface{})
		// Check all keys from a
		for key, aChild := range aVal {
			bChild, exists := bVal[key]
			if !exists {
				return false
			}
			childPath := path + "." + key
			if !e.deepEqual(aChild, bChild, childPath) {
				return false
			}
		}
		// Check for extra keys in b
		for key := range bVal {
			if _, exists := aVal[key]; !exists {
				return false
			}
		}
		return true

	case []interface{}:
		bVal := b.([]interface{})
		if len(aVal) != len(bVal) {
			return false
		}
		for i := range aVal {
			childPath := fmt.Sprintf("%s[%d]", path, i)
			if !e.deepEqual(aVal[i], bVal[i], childPath) {
				return false
			}
		}
		return true

	case float64:
		bVal := b.(float64)
		return e.numericEqual(aVal, bVal)

	case string:
		bVal := b.(string)
		if e.stringNormalize {
			return normalizeString(aVal) == normalizeString(bVal)
		}
		return aVal == bVal

	case bool:
		return aVal == b.(bool)

	default:
		return reflect.DeepEqual(a, b)
	}
}

// numericEqual checks if two numbers are within tolerance
func (e *Engine) numericEqual(a, b float64) bool {
	// Handle exact equality (including zeros)
	if a == b {
		return true
	}

	// Calculate relative difference
	// Use the larger absolute value as the denominator
	maxAbs := math.Max(math.Abs(a), math.Abs(b))
	if maxAbs == 0 {
		return true
	}

	diff := math.Abs(a-b) / maxAbs
	return diff <= e.numericTolerance
}

// toFloat64 attempts to convert a value to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}

// normalizeString normalizes a string for comparison
func normalizeString(s string) string {
	// Convert to lowercase
	s = lower(s)
	// Normalize whitespace
	s = collapseWhitespace(s)
	return s
}

// lower is a simple lowercase function
func lower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// collapseWhitespace collapses multiple whitespace to single space
func collapseWhitespace(s string) string {
	result := make([]byte, 0, len(s))
	inSpace := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !inSpace {
				result = append(result, ' ')
				inSpace = true
			}
		} else {
			result = append(result, c)
			inSpace = false
		}
	}
	// Trim leading/trailing space
	if len(result) > 0 && result[0] == ' ' {
		result = result[1:]
	}
	if len(result) > 0 && result[len(result)-1] == ' ' {
		result = result[:len(result)-1]
	}
	return string(result)
}

// =============================================================================
// DISAGREEMENT DETECTION
// =============================================================================

// findDisagreements identifies specific fields where providers disagree
func (e *Engine) findDisagreements(results []llm.ExtractionResult) []Disagreement {
	if len(results) < 2 {
		return nil
	}

	disagreements := make([]Disagreement, 0)

	// Compare first result with all others
	first := results[0]
	for i := 1; i < len(results); i++ {
		other := results[i]
		diffs := e.findDifferences(first.ExtractedData, other.ExtractedData, "")
		for _, diff := range diffs {
			disagreements = append(disagreements, Disagreement{
				Field:     diff.path,
				Provider1: first.Provider,
				Value1:    diff.value1,
				Provider2: other.Provider,
				Value2:    diff.value2,
				Severity:  e.assessSeverity(diff),
			})
		}
	}

	return disagreements
}

type fieldDiff struct {
	path   string
	value1 interface{}
	value2 interface{}
}

// findDifferences recursively finds differences between two values
func (e *Engine) findDifferences(a, b interface{}, path string) []fieldDiff {
	if e.deepEqual(a, b, path) {
		return nil
	}

	diffs := make([]fieldDiff, 0)

	// Handle nil
	if a == nil || b == nil {
		return []fieldDiff{{path: path, value1: a, value2: b}}
	}

	// Convert to maps for comparison
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)

	var aMap, bMap map[string]interface{}
	if err := json.Unmarshal(aJSON, &aMap); err != nil {
		return []fieldDiff{{path: path, value1: a, value2: b}}
	}
	if err := json.Unmarshal(bJSON, &bMap); err != nil {
		return []fieldDiff{{path: path, value1: a, value2: b}}
	}

	// Find differences in each field
	allKeys := make(map[string]bool)
	for k := range aMap {
		allKeys[k] = true
	}
	for k := range bMap {
		allKeys[k] = true
	}

	for key := range allKeys {
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}

		aVal := aMap[key]
		bVal := bMap[key]

		if !e.deepEqual(aVal, bVal, childPath) {
			diffs = append(diffs, fieldDiff{
				path:   childPath,
				value1: aVal,
				value2: bVal,
			})
		}
	}

	return diffs
}

// assessSeverity determines the severity of a disagreement
func (e *Engine) assessSeverity(diff fieldDiff) DisagreementSeverity {
	// Check for numeric values with large differences
	a, aNum := toFloat64(diff.value1)
	b, bNum := toFloat64(diff.value2)

	if aNum && bNum {
		// Calculate relative difference
		maxAbs := math.Max(math.Abs(a), math.Abs(b))
		if maxAbs > 0 {
			relDiff := math.Abs(a-b) / maxAbs
			if relDiff > 0.2 { // More than 20% difference
				return SeverityCritical
			}
		}
	}

	// Check for boolean differences (always critical for clinical data)
	if _, ok := diff.value1.(bool); ok {
		return SeverityCritical
	}

	// Check for action/recommendation differences
	pathLower := lower(diff.path)
	if contains(pathLower, "action") ||
		contains(pathLower, "contraindicated") ||
		contains(pathLower, "avoid") ||
		contains(pathLower, "severity") {
		return SeverityCritical
	}

	return SeverityMinor
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// AddProvider adds a provider to the engine
func (e *Engine) AddProvider(provider llm.Provider) {
	e.providers = append(e.providers, provider)
}

// ProviderCount returns the number of registered providers
func (e *Engine) ProviderCount() int {
	return len(e.providers)
}

// SetMinAgreement updates the minimum agreement threshold
func (e *Engine) SetMinAgreement(n int) {
	e.minAgreement = n
}

// SetTimeout updates the timeout duration
func (e *Engine) SetTimeout(d time.Duration) {
	e.timeout = d
}

// SetNumericTolerance updates the numeric comparison tolerance
func (e *Engine) SetNumericTolerance(t float64) {
	e.numericTolerance = t
}
