package fda

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// Client provides access to FDA DailyMed API for drug information retrieval
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
	rateLimitMs int
}

// ClientConfig holds FDA client configuration
type ClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	RateLimitMs int
}

// DefaultClientConfig returns default FDA client configuration
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseURL:     "https://dailymed.nlm.nih.gov/dailymed/services/v2",
		Timeout:     60 * time.Second,
		RateLimitMs: 100,
	}
}

// NewClient creates a new FDA DailyMed client
func NewClient(log *logrus.Entry) *Client {
	cfg := DefaultClientConfig()
	return NewClientWithConfig(cfg, log)
}

// NewClientWithConfig creates a client with custom configuration
func NewClientWithConfig(cfg ClientConfig, log *logrus.Entry) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		log:         log.WithField("component", "fda-client"),
		rateLimitMs: cfg.RateLimitMs,
	}
}

// =============================================================================
// DATA STRUCTURES
// =============================================================================

// DailyMedSearchResult represents search results from DailyMed API
type DailyMedSearchResult struct {
	Data []DrugListItem `json:"data"`
	Metadata struct {
		TotalElements int `json:"total_elements"`
		TotalPages    int `json:"total_pages"`
		CurrentPage   int `json:"current_page"`
		PageSize      int `json:"pagesize"`
	} `json:"metadata"`
}

// DrugListItem represents a single drug in search results
type DrugListItem struct {
	SetID         string `json:"setid"`
	Title         string `json:"title"`
	PublishedDate string `json:"published_date"`
	ProductNDC    string `json:"product_ndc"`
	VersionNumber string `json:"version_number"`
}

// NDCInfo represents NDC-specific drug information
type NDCInfo struct {
	NDC                string   `json:"ndc"`
	ProductNDC         string   `json:"product_ndc"`
	GenericName        string   `json:"generic_name"`
	BrandName          string   `json:"brand_name"`
	DosageForm         string   `json:"dosage_form"`
	Route              string   `json:"route"`
	Strength           string   `json:"active_numerator_strength"`
	StrengthUnit       string   `json:"active_ingred_unit"`
	PharmClasses       []string `json:"pharm_class"`
	Manufacturer       string   `json:"labeler_name"`
	MarketingStatus    string   `json:"marketing_status"`
	ApplicationNumber  string   `json:"application_number"`
}

// =============================================================================
// SEARCH OPERATIONS
// =============================================================================

// SearchDrugs searches for drugs by name
func (c *Client) SearchDrugs(ctx context.Context, drugName string, page, pageSize int) (*DailyMedSearchResult, error) {
	endpoint := fmt.Sprintf("%s/spls.json?drug_name=%s&page=%d&pagesize=%d",
		c.baseURL, url.QueryEscape(drugName), page, pageSize)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result DailyMedSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetAllDrugSetIDs retrieves all drug SetIDs (paginated)
// This is the main entry point for full formulary ingestion
func (c *Client) GetAllDrugSetIDs(ctx context.Context, pageSize int) ([]string, error) {
	var allSetIDs []string
	page := 1

	c.log.Info("Starting to fetch all drug SetIDs from FDA DailyMed")

	for {
		select {
		case <-ctx.Done():
			return allSetIDs, ctx.Err()
		default:
		}

		endpoint := fmt.Sprintf("%s/spls.json?page=%d&pagesize=%d", c.baseURL, page, pageSize)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed on page %d: %w", page, err)
		}

		var result DailyMedSearchResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response on page %d: %w", page, err)
		}
		resp.Body.Close()

		for _, drug := range result.Data {
			allSetIDs = append(allSetIDs, drug.SetID)
		}

		c.log.WithFields(logrus.Fields{
			"page":          page,
			"total_pages":   result.Metadata.TotalPages,
			"page_items":    len(result.Data),
			"total_collected": len(allSetIDs),
		}).Debug("Fetched drug SetIDs page")

		if page >= result.Metadata.TotalPages || len(result.Data) == 0 {
			break
		}
		page++

		// Rate limiting to avoid overwhelming FDA servers
		if c.rateLimitMs > 0 {
			time.Sleep(time.Duration(c.rateLimitMs) * time.Millisecond)
		}
	}

	c.log.WithField("total_drugs", len(allSetIDs)).Info("Completed fetching all drug SetIDs")
	return allSetIDs, nil
}

// =============================================================================
// SPL DOCUMENT RETRIEVAL
// =============================================================================

// FetchSPL retrieves the full SPL XML document for a drug by SetID
func (c *Client) FetchSPL(ctx context.Context, setID string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/spls/%s.xml", c.baseURL, setID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Drug not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// FetchSPLJSON retrieves SPL document in JSON format
func (c *Client) FetchSPLJSON(ctx context.Context, setID string) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("%s/spls/%s.json", c.baseURL, setID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// =============================================================================
// NDC OPERATIONS
// =============================================================================

// FetchNDCInfo fetches NDC-specific drug information
func (c *Client) FetchNDCInfo(ctx context.Context, ndc string) (*NDCInfo, error) {
	endpoint := fmt.Sprintf("%s/ndcs/%s.json", c.baseURL, ndc)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result NDCInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetNDCsForSetID retrieves all NDC codes associated with a SetID
func (c *Client) GetNDCsForSetID(ctx context.Context, setID string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/spls/%s/ndcs.json", c.baseURL, setID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			NDC string `json:"ndc"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	ndcs := make([]string, len(result.Data))
	for i, item := range result.Data {
		ndcs[i] = item.NDC
	}

	return ndcs, nil
}

// =============================================================================
// BATCH OPERATIONS
// =============================================================================

// BatchFetchSPLs fetches multiple SPL documents with rate limiting
func (c *Client) BatchFetchSPLs(ctx context.Context, setIDs []string) (map[string][]byte, error) {
	results := make(map[string][]byte)

	for _, setID := range setIDs {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		spl, err := c.FetchSPL(ctx, setID)
		if err != nil {
			c.log.WithError(err).WithField("set_id", setID).Warn("Failed to fetch SPL")
			continue
		}
		if spl != nil {
			results[setID] = spl
		}

		// Rate limiting
		if c.rateLimitMs > 0 {
			time.Sleep(time.Duration(c.rateLimitMs) * time.Millisecond)
		}
	}

	return results, nil
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// Health checks FDA DailyMed API availability
func (c *Client) Health(ctx context.Context) error {
	// Use a simple search to verify API is responding
	endpoint := fmt.Sprintf("%s/spls.json?pagesize=1", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("FDA DailyMed health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FDA DailyMed unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
