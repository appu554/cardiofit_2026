// Package llm provides the Claude Provider implementation.
//
// Phase 3c.1: Claude Provider for Clinical Fact Extraction
// Authority Level: GAP-FILLER ONLY
//
// Claude 3 Opus/Sonnet are used for structured clinical data extraction
// when structured sources (tables, APIs) are not available.
//
// KEY FEATURES:
// - Structured output with JSON schema enforcement
// - Explicit citation extraction from source text
// - Low temperature for deterministic clinical extractions
// - Detailed token and cost tracking
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// CLAUDE PROVIDER
// =============================================================================

// ClaudeProvider implements Provider for Anthropic Claude
type ClaudeProvider struct {
	apiKey         string
	model          string // "claude-3-opus-20240229", "claude-3-sonnet-20240229"
	httpClient     *http.Client
	baseURL        string
	version        string
	maxContextSize int
}

// ClaudeConfig contains configuration for the Claude provider
type ClaudeConfig struct {
	// APIKey is the Anthropic API key (required)
	APIKey string

	// Model is the Claude model to use
	// Recommended: "claude-3-opus-20240229" for highest accuracy
	//              "claude-3-sonnet-20240229" for balanced speed/accuracy
	Model string

	// BaseURL is the Anthropic API endpoint (optional, defaults to production)
	BaseURL string

	// Timeout is the HTTP request timeout
	Timeout time.Duration
}

// DefaultClaudeConfig returns a default configuration
func DefaultClaudeConfig() ClaudeConfig {
	return ClaudeConfig{
		Model:   "claude-3-opus-20240229",
		BaseURL: "https://api.anthropic.com",
		Timeout: 60 * time.Second,
	}
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(config ClaudeConfig) *ClaudeProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	maxContext := 200000 // Claude 3 context window
	if strings.Contains(config.Model, "sonnet") {
		maxContext = 200000
	}

	return &ClaudeProvider{
		apiKey:         config.APIKey,
		model:          config.Model,
		baseURL:        config.BaseURL,
		httpClient:     &http.Client{Timeout: config.Timeout},
		version:        "1.0.0",
		maxContextSize: maxContext,
	}
}

// =============================================================================
// PROVIDER INTERFACE IMPLEMENTATION
// =============================================================================

// Name returns the provider name
func (c *ClaudeProvider) Name() string {
	return fmt.Sprintf("claude-%s", c.model)
}

// Version returns the provider implementation version
func (c *ClaudeProvider) Version() string {
	return c.version
}

// SupportsStructuredOutput returns true (Claude supports JSON mode)
func (c *ClaudeProvider) SupportsStructuredOutput() bool {
	return true
}

// MaxTokens returns the maximum context window
func (c *ClaudeProvider) MaxTokens() int {
	return c.maxContextSize
}

// CostPerToken returns the cost per token in USD
func (c *ClaudeProvider) CostPerToken() float64 {
	// Claude 3 Opus pricing (as of 2024)
	if strings.Contains(c.model, "opus") {
		return 0.000015 // $15 per million input tokens
	}
	// Claude 3 Sonnet pricing
	return 0.000003 // $3 per million input tokens
}

// Extract processes source text and extracts structured facts
func (c *ClaudeProvider) Extract(ctx context.Context, req *ExtractionRequest) (*ExtractionResult, error) {
	startTime := time.Now()
	result := NewExtractionResult(c.Name(), req.FactType)
	result.RequestID = req.RequestID
	result.ProviderVersion = c.model

	// Build the prompt
	prompt, err := c.buildPrompt(req)
	if err != nil {
		result.SetError(err)
		return result, err
	}

	// Make the API call with retries
	var lastErr error
	for attempt := 0; attempt <= req.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
			result.RetryCount = attempt
		}

		response, err := c.callAPI(ctx, prompt, req.Temperature)
		if err != nil {
			lastErr = err
			continue
		}

		// Parse the response
		if err := c.parseResponse(response, result, req); err != nil {
			lastErr = err
			continue
		}

		// Success!
		result.MarkSuccess()
		result.Latency = time.Since(startTime)
		return result, nil
	}

	// All retries failed
	result.SetError(fmt.Errorf("all %d attempts failed: %w", req.MaxRetries+1, lastErr))
	result.Latency = time.Since(startTime)
	return result, lastErr
}

// =============================================================================
// PROMPT BUILDING
// =============================================================================

func (c *ClaudeProvider) buildPrompt(req *ExtractionRequest) (string, error) {
	schemaJSON := string(req.Schema.JSONSchema)

	var contextInfo string
	if req.DrugContext != nil {
		contextInfo = fmt.Sprintf(`
Drug Context:
- Drug Name: %s
- Generic Name: %s
- RxCUI: %s
- Drug Class: %s
`, req.DrugContext.DrugName, req.DrugContext.GenericName,
			req.DrugContext.RxCUI, req.DrugContext.DrugClass)
	}

	// Build the extraction prompt
	prompt := fmt.Sprintf(`You are a clinical pharmacology expert extracting structured drug information from FDA Structured Product Labels (SPL).

TASK: Extract %s information from the following source text.

%s
OUTPUT REQUIREMENTS:
1. Return ONLY valid JSON matching this schema:
%s

2. Include citations from the source text for EVERY extracted value.
3. Be conservative - if information is unclear or ambiguous, set confidence lower.
4. Do not infer or guess - only extract explicitly stated information.
5. If the source text does not contain the requested information, return null values.

CONFIDENCE GUIDELINES:
- 0.95-1.0: Information is explicitly stated with exact values
- 0.80-0.94: Information is stated but requires minor interpretation
- 0.60-0.79: Information requires significant interpretation
- Below 0.60: Information is implied or uncertain (consider not extracting)

SOURCE TEXT:
<source>
%s
</source>

Return your response as a JSON object with this structure:
{
  "extractedData": <extracted data matching schema>,
  "confidence": <overall confidence 0.0-1.0>,
  "confidenceExplanation": "<why this confidence level>",
  "citations": [
    {
      "quotedText": "<exact quote from source>",
      "supportsFact": "<which field this supports>",
      "confidence": <citation-specific confidence>
    }
  ],
  "warnings": ["<any extraction warnings>"]
}

IMPORTANT: Return ONLY the JSON response, no other text.`,
		req.FactType,
		contextInfo,
		schemaJSON,
		req.SourceText,
	)

	return prompt, nil
}

// =============================================================================
// API CALLS
// =============================================================================

// claudeRequest is the Anthropic API request structure
type claudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
	Messages    []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// claudeResponse is the Anthropic API response structure
type claudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (c *ClaudeProvider) callAPI(ctx context.Context, prompt string, temperature float64) (*claudeResponse, error) {
	reqBody := claudeRequest{
		Model:       c.model,
		MaxTokens:   4096,
		Temperature: temperature,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var response claudeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &response, nil
}

// =============================================================================
// RESPONSE PARSING
// =============================================================================

// claudeExtraction represents the structured extraction response
type claudeExtraction struct {
	ExtractedData         interface{}         `json:"extractedData"`
	Confidence            float64             `json:"confidence"`
	ConfidenceExplanation string              `json:"confidenceExplanation"`
	Citations             []claudeCitation    `json:"citations"`
	Warnings              []string            `json:"warnings"`
}

type claudeCitation struct {
	QuotedText   string  `json:"quotedText"`
	SupportsFact string  `json:"supportsFact"`
	Confidence   float64 `json:"confidence"`
}

func (c *ClaudeProvider) parseResponse(response *claudeResponse, result *ExtractionResult, req *ExtractionRequest) error {
	// Extract the text content
	if len(response.Content) == 0 {
		return fmt.Errorf("empty response content")
	}

	rawText := response.Content[0].Text
	result.RawResponse = rawText

	// Record token usage
	result.TokensUsed = TokenUsage{
		PromptTokens:     response.Usage.InputTokens,
		CompletionTokens: response.Usage.OutputTokens,
		TotalTokens:      response.Usage.InputTokens + response.Usage.OutputTokens,
	}

	// Calculate cost
	inputCost := 15.0  // $15 per million for Opus
	outputCost := 75.0 // $75 per million for Opus output
	if strings.Contains(c.model, "sonnet") {
		inputCost = 3.0
		outputCost = 15.0
	}
	result.CalculateCost(inputCost, outputCost)

	// Parse the JSON response
	// First, try to extract JSON from the response (handle markdown code blocks)
	jsonStr := extractJSON(rawText)

	var extraction claudeExtraction
	if err := json.Unmarshal([]byte(jsonStr), &extraction); err != nil {
		return fmt.Errorf("parsing extraction JSON: %w (raw: %s)", err, jsonStr[:min(200, len(jsonStr))])
	}

	// Set the extracted data
	if err := result.SetExtractedData(extraction.ExtractedData); err != nil {
		return fmt.Errorf("setting extracted data: %w", err)
	}

	// Set confidence
	result.Confidence = extraction.Confidence
	result.ConfidenceExplanation = extraction.ConfidenceExplanation

	// Add citations
	for _, cit := range extraction.Citations {
		result.AddCitation(0, 0, cit.QuotedText, cit.SupportsFact, cit.Confidence)
	}

	// Add warnings
	for _, warning := range extraction.Warnings {
		result.AddWarning(warning)
	}

	return nil
}

// extractJSON extracts JSON from text that may contain markdown code blocks
func extractJSON(text string) string {
	// Remove markdown code blocks if present
	text = strings.TrimSpace(text)

	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		text = strings.TrimSuffix(text, "```")
		text = strings.TrimSpace(text)
	}

	return text
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
