// Package llm provides the GPT-4 Provider implementation.
//
// Phase 3c.1: GPT-4 Provider for Clinical Fact Extraction
// Authority Level: GAP-FILLER ONLY
//
// GPT-4 Turbo/GPT-4o are used as part of the consensus extraction system.
// This provider works alongside Claude to achieve 2-of-3 agreement.
//
// KEY FEATURES:
// - JSON mode for structured output
// - Function calling for schema enforcement
// - Explicit citation extraction from source text
// - OpenAI API compatibility
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
// GPT-4 PROVIDER
// =============================================================================

// GPT4Provider implements Provider for OpenAI GPT-4
type GPT4Provider struct {
	apiKey         string
	model          string // "gpt-4-turbo-preview", "gpt-4o", "gpt-4"
	httpClient     *http.Client
	baseURL        string
	version        string
	maxContextSize int
	orgID          string // Optional organization ID
}

// GPT4Config contains configuration for the GPT-4 provider
type GPT4Config struct {
	// APIKey is the OpenAI API key (required)
	APIKey string

	// Model is the GPT-4 model to use
	// Recommended: "gpt-4-turbo-preview" for JSON mode
	//              "gpt-4o" for latest multimodal
	//              "gpt-4" for original GPT-4
	Model string

	// BaseURL is the OpenAI API endpoint (optional, defaults to production)
	BaseURL string

	// Timeout is the HTTP request timeout
	Timeout time.Duration

	// OrgID is the optional organization ID
	OrgID string
}

// DefaultGPT4Config returns a default configuration
func DefaultGPT4Config() GPT4Config {
	return GPT4Config{
		Model:   "gpt-4-turbo-preview",
		BaseURL: "https://api.openai.com",
		Timeout: 60 * time.Second,
	}
}

// NewGPT4Provider creates a new GPT-4 provider
func NewGPT4Provider(config GPT4Config) *GPT4Provider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	maxContext := 128000 // GPT-4 Turbo context window
	if config.Model == "gpt-4" {
		maxContext = 8192
	} else if strings.Contains(config.Model, "gpt-4o") {
		maxContext = 128000
	}

	return &GPT4Provider{
		apiKey:         config.APIKey,
		model:          config.Model,
		baseURL:        config.BaseURL,
		httpClient:     &http.Client{Timeout: config.Timeout},
		version:        "1.0.0",
		maxContextSize: maxContext,
		orgID:          config.OrgID,
	}
}

// =============================================================================
// PROVIDER INTERFACE IMPLEMENTATION
// =============================================================================

// Name returns the provider name
func (g *GPT4Provider) Name() string {
	return fmt.Sprintf("openai-%s", g.model)
}

// Version returns the provider implementation version
func (g *GPT4Provider) Version() string {
	return g.version
}

// SupportsStructuredOutput returns true (GPT-4 supports JSON mode)
func (g *GPT4Provider) SupportsStructuredOutput() bool {
	// GPT-4 Turbo and GPT-4o support JSON mode
	return strings.Contains(g.model, "turbo") || strings.Contains(g.model, "gpt-4o")
}

// MaxTokens returns the maximum context window
func (g *GPT4Provider) MaxTokens() int {
	return g.maxContextSize
}

// CostPerToken returns the cost per token in USD
func (g *GPT4Provider) CostPerToken() float64 {
	// GPT-4 Turbo pricing (as of 2024)
	if strings.Contains(g.model, "turbo") {
		return 0.00001 // $10 per million input tokens
	}
	// GPT-4o pricing
	if strings.Contains(g.model, "gpt-4o") {
		return 0.000005 // $5 per million input tokens
	}
	// Original GPT-4 pricing
	return 0.00003 // $30 per million input tokens
}

// Extract processes source text and extracts structured facts
func (g *GPT4Provider) Extract(ctx context.Context, req *ExtractionRequest) (*ExtractionResult, error) {
	startTime := time.Now()
	result := NewExtractionResult(g.Name(), req.FactType)
	result.RequestID = req.RequestID
	result.ProviderVersion = g.model

	// Build the prompt
	systemPrompt, userPrompt, err := g.buildPrompts(req)
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

		response, err := g.callAPI(ctx, systemPrompt, userPrompt, req.Temperature)
		if err != nil {
			lastErr = err
			continue
		}

		// Parse the response
		if err := g.parseResponse(response, result, req); err != nil {
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

func (g *GPT4Provider) buildPrompts(req *ExtractionRequest) (string, string, error) {
	schemaJSON := string(req.Schema.JSONSchema)

	// System prompt
	systemPrompt := `You are a clinical pharmacology expert extracting structured drug information from FDA Structured Product Labels (SPL).

Your role is to:
1. Extract precise, clinically accurate information from drug labels
2. Provide explicit citations from the source text
3. Be conservative - never infer beyond what is explicitly stated
4. Maintain high accuracy for patient safety

OUTPUT FORMAT: You must respond with valid JSON only, no other text.

CONFIDENCE GUIDELINES:
- 0.95-1.0: Information is explicitly stated with exact values
- 0.80-0.94: Information is stated but requires minor interpretation
- 0.60-0.79: Information requires significant interpretation
- Below 0.60: Information is implied or uncertain (consider not extracting)`

	// Build context info
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

	// User prompt
	userPrompt := fmt.Sprintf(`Extract %s information from the following FDA drug label section.

%s
REQUIRED OUTPUT SCHEMA:
%s

SOURCE TEXT:
<source>
%s
</source>

Return your response as a JSON object with this structure:
{
  "extractedData": <extracted data matching the schema above>,
  "confidence": <overall confidence 0.0-1.0>,
  "confidenceExplanation": "<brief explanation of confidence level>",
  "citations": [
    {
      "quotedText": "<exact quote from source>",
      "supportsFact": "<which field this supports>",
      "confidence": <citation-specific confidence>
    }
  ],
  "warnings": ["<any extraction warnings>"]
}

IMPORTANT: Return ONLY valid JSON, no markdown, no explanatory text.`,
		req.FactType,
		contextInfo,
		schemaJSON,
		req.SourceText,
	)

	return systemPrompt, userPrompt, nil
}

// =============================================================================
// API CALLS
// =============================================================================

// gpt4Request is the OpenAI API request structure
type gpt4Request struct {
	Model          string        `json:"model"`
	Messages       []gpt4Message `json:"messages"`
	MaxTokens      int           `json:"max_tokens"`
	Temperature    float64       `json:"temperature"`
	ResponseFormat *gpt4Format   `json:"response_format,omitempty"`
}

type gpt4Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type gpt4Format struct {
	Type string `json:"type"`
}

// gpt4Response is the OpenAI API response structure
type gpt4Response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      gpt4Message `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func (g *GPT4Provider) callAPI(ctx context.Context, systemPrompt, userPrompt string, temperature float64) (*gpt4Response, error) {
	reqBody := gpt4Request{
		Model: g.model,
		Messages: []gpt4Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   4096,
		Temperature: temperature,
	}

	// Enable JSON mode for supported models
	if g.SupportsStructuredOutput() {
		reqBody.ResponseFormat = &gpt4Format{Type: "json_object"}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.baseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	if g.orgID != "" {
		req.Header.Set("OpenAI-Organization", g.orgID)
	}

	resp, err := g.httpClient.Do(req)
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

	var response gpt4Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &response, nil
}

// =============================================================================
// RESPONSE PARSING
// =============================================================================

// gpt4Extraction represents the structured extraction response
type gpt4Extraction struct {
	ExtractedData         interface{}      `json:"extractedData"`
	Confidence            float64          `json:"confidence"`
	ConfidenceExplanation string           `json:"confidenceExplanation"`
	Citations             []gpt4Citation   `json:"citations"`
	Warnings              []string         `json:"warnings"`
}

type gpt4Citation struct {
	QuotedText   string  `json:"quotedText"`
	SupportsFact string  `json:"supportsFact"`
	Confidence   float64 `json:"confidence"`
}

func (g *GPT4Provider) parseResponse(response *gpt4Response, result *ExtractionResult, req *ExtractionRequest) error {
	// Extract the text content
	if len(response.Choices) == 0 {
		return fmt.Errorf("no choices in response")
	}

	rawText := response.Choices[0].Message.Content
	result.RawResponse = rawText

	// Record token usage
	result.TokensUsed = TokenUsage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}

	// Calculate cost based on model
	inputCost := 10.0  // Default $10 per million for GPT-4 Turbo
	outputCost := 30.0 // $30 per million for GPT-4 Turbo output

	if strings.Contains(g.model, "gpt-4o") {
		inputCost = 5.0
		outputCost = 15.0
	} else if g.model == "gpt-4" {
		inputCost = 30.0
		outputCost = 60.0
	}
	result.CalculateCost(inputCost, outputCost)

	// Parse the JSON response
	// Handle potential markdown code blocks
	jsonStr := extractJSONFromText(rawText)

	var extraction gpt4Extraction
	if err := json.Unmarshal([]byte(jsonStr), &extraction); err != nil {
		return fmt.Errorf("parsing extraction JSON: %w (raw: %s)", err, truncateString(jsonStr, 200))
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

// extractJSONFromText extracts JSON from text that may contain markdown code blocks
func extractJSONFromText(text string) string {
	text = strings.TrimSpace(text)

	// Remove markdown code blocks if present
	if strings.HasPrefix(text, "```json") {
		text = strings.TrimPrefix(text, "```json")
		if idx := strings.LastIndex(text, "```"); idx > 0 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	} else if strings.HasPrefix(text, "```") {
		text = strings.TrimPrefix(text, "```")
		if idx := strings.LastIndex(text, "```"); idx > 0 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	return text
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
