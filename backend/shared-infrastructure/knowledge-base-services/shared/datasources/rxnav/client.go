// Package rxnav provides a client for the NLM RxNav REST API.
// RxNav is a browser for drugs in RxNorm, providing access to drug names,
// codes, relationships, and interactions.
//
// API Documentation: https://lhncbc.nlm.nih.gov/RxNav/APIs/RxNormAPIs.html
package rxnav

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources"
)

// =============================================================================
// CLIENT CONFIGURATION
// =============================================================================

// Config holds configuration for the RxNav client
type Config struct {
	// BaseURL is the RxNav API base URL
	BaseURL string

	// Timeout for HTTP requests
	Timeout time.Duration

	// MaxRetries for failed requests
	MaxRetries int

	// RetryDelay between retries
	RetryDelay time.Duration

	// RateLimitPerSecond limits requests per second (0 = unlimited)
	RateLimitPerSecond int

	// Cache is optional caching layer
	Cache datasources.Cache

	// CacheTTL for cached responses
	CacheTTL time.Duration

	// HTTPClient is the HTTP client to use (optional)
	HTTPClient *http.Client

	// Logger for logging
	Logger *logrus.Entry
}

// DefaultConfig returns default configuration using public RxNav API
func DefaultConfig() Config {
	return Config{
		BaseURL:            "https://rxnav.nlm.nih.gov/REST",
		Timeout:            30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
		RateLimitPerSecond: 20, // NLM recommends max 20 req/sec
		CacheTTL:           24 * time.Hour,
	}
}

// LocalConfig returns configuration for RxNav-in-a-Box (local Docker instance)
// This provides unlimited local API calls without rate limits
// Start with: cd rxnav-in-a-box-* && docker-compose up -d
func LocalConfig() Config {
	return Config{
		BaseURL:            "http://localhost:4000/REST",
		Timeout:            30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         500 * time.Millisecond,
		RateLimitPerSecond: 0, // No rate limit for local instance
		CacheTTL:           24 * time.Hour,
	}
}

// LocalConfigWithPort returns configuration for RxNav-in-a-Box on custom port
func LocalConfigWithPort(port int) Config {
	return Config{
		BaseURL:            fmt.Sprintf("http://localhost:%d/REST", port),
		Timeout:            30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         500 * time.Millisecond,
		RateLimitPerSecond: 0,
		CacheTTL:           24 * time.Hour,
	}
}

// =============================================================================
// CLIENT IMPLEMENTATION
// =============================================================================

// Client implements the RxNavClient interface
type Client struct {
	config     Config
	httpClient *http.Client
	log        *logrus.Entry
	cache      datasources.Cache

	// Rate limiting
	rateLimiter chan struct{}
	lastRequest time.Time
	mu          sync.Mutex
}

// NewClient creates a new RxNav client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://rxnav.nlm.nih.gov/REST"
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	log := config.Logger
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}

	c := &Client{
		config:     config,
		httpClient: httpClient,
		log:        log.WithField("datasource", "rxnav"),
		cache:      config.Cache,
	}

	// Initialize rate limiter
	if config.RateLimitPerSecond > 0 {
		c.rateLimiter = make(chan struct{}, config.RateLimitPerSecond)
		for i := 0; i < config.RateLimitPerSecond; i++ {
			c.rateLimiter <- struct{}{}
		}
		go c.refillRateLimiter()
	}

	return c
}

// =============================================================================
// DATASOURCE INTERFACE
// =============================================================================

func (c *Client) Name() string {
	return "rxnav"
}

func (c *Client) HealthCheck(ctx context.Context) error {
	// Test with a simple known drug
	_, err := c.GetRxCUIByName(ctx, "aspirin")
	return err
}

func (c *Client) Close() error {
	if c.cache != nil {
		return c.cache.Close()
	}
	return nil
}

// =============================================================================
// DRUG LOOKUP OPERATIONS
// =============================================================================

// GetRxCUIByName finds the RxCUI for a drug name
func (c *Client) GetRxCUIByName(ctx context.Context, drugName string) (string, error) {
	path := fmt.Sprintf("/rxcui.json?name=%s", url.QueryEscape(drugName))

	var response struct {
		IdGroup struct {
			RxnormID []string `json:"rxnormId"`
		} `json:"idGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return "", err
	}

	if len(response.IdGroup.RxnormID) == 0 {
		return "", fmt.Errorf("no RxCUI found for drug: %s", drugName)
	}

	return response.IdGroup.RxnormID[0], nil
}

// GetRxCUIByNDC finds the RxCUI for a National Drug Code
func (c *Client) GetRxCUIByNDC(ctx context.Context, ndc string) (string, error) {
	path := fmt.Sprintf("/rxcui.json?idtype=NDC&id=%s", url.QueryEscape(ndc))

	var response struct {
		IdGroup struct {
			RxnormID []string `json:"rxnormId"`
		} `json:"idGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return "", err
	}

	if len(response.IdGroup.RxnormID) == 0 {
		return "", fmt.Errorf("no RxCUI found for NDC: %s", ndc)
	}

	return response.IdGroup.RxnormID[0], nil
}

// GetDrugByRxCUI retrieves drug details by RxCUI
func (c *Client) GetDrugByRxCUI(ctx context.Context, rxcui string) (*datasources.RxNormDrug, error) {
	path := fmt.Sprintf("/rxcui/%s/properties.json", rxcui)

	var response struct {
		Properties struct {
			RxCUI    string `json:"rxcui"`
			Name     string `json:"name"`
			Synonym  string `json:"synonym"`
			TTY      string `json:"tty"`
			Language string `json:"language"`
			Suppress string `json:"suppress"`
		} `json:"properties"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	drug := &datasources.RxNormDrug{
		RxCUI:    response.Properties.RxCUI,
		Name:     response.Properties.Name,
		Synonym:  response.Properties.Synonym,
		TTY:      response.Properties.TTY,
		Language: response.Properties.Language,
		Suppress: response.Properties.Suppress,
	}

	// Get additional details
	ingredients, _ := c.GetIngredients(ctx, rxcui)
	for _, ing := range ingredients {
		drug.Ingredients = append(drug.Ingredients, ing.Name)
	}

	return drug, nil
}

// SearchDrugs searches for drugs matching a query
func (c *Client) SearchDrugs(ctx context.Context, query string, limit int) ([]datasources.RxNormDrug, error) {
	path := fmt.Sprintf("/approximateTerm.json?term=%s&maxEntries=%d", url.QueryEscape(query), limit)

	var response struct {
		ApproximateGroup struct {
			Candidate []struct {
				RxCUI string `json:"rxcui"`
				Name  string `json:"name"`
				Score string `json:"score"`
			} `json:"candidate"`
		} `json:"approximateGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	var drugs []datasources.RxNormDrug
	for _, cand := range response.ApproximateGroup.Candidate {
		drugs = append(drugs, datasources.RxNormDrug{
			RxCUI: cand.RxCUI,
			Name:  cand.Name,
		})
	}

	return drugs, nil
}

// =============================================================================
// RELATIONSHIP OPERATIONS
// =============================================================================

// GetRelatedByType finds related concepts by relationship type
func (c *Client) GetRelatedByType(ctx context.Context, rxcui string, relType datasources.RxNormRelationType) ([]datasources.RxNormConcept, error) {
	path := fmt.Sprintf("/rxcui/%s/related.json?tty=%s", rxcui, string(relType))

	var response struct {
		RelatedGroup struct {
			ConceptGroup []struct {
				TTY           string `json:"tty"`
				ConceptProperties []struct {
					RxCUI string `json:"rxcui"`
					Name  string `json:"name"`
					TTY   string `json:"tty"`
				} `json:"conceptProperties"`
			} `json:"conceptGroup"`
		} `json:"relatedGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	var concepts []datasources.RxNormConcept
	for _, group := range response.RelatedGroup.ConceptGroup {
		for _, prop := range group.ConceptProperties {
			concepts = append(concepts, datasources.RxNormConcept{
				RxCUI: prop.RxCUI,
				Name:  prop.Name,
				TTY:   prop.TTY,
			})
		}
	}

	return concepts, nil
}

// GetAllRelated retrieves all relationships for a drug
func (c *Client) GetAllRelated(ctx context.Context, rxcui string) (*datasources.RxNormRelationships, error) {
	path := fmt.Sprintf("/rxcui/%s/allrelated.json", rxcui)

	var response struct {
		AllRelatedGroup struct {
			ConceptGroup []struct {
				TTY               string `json:"tty"`
				ConceptProperties []struct {
					RxCUI string `json:"rxcui"`
					Name  string `json:"name"`
					TTY   string `json:"tty"`
				} `json:"conceptProperties"`
			} `json:"conceptGroup"`
		} `json:"allRelatedGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	rel := &datasources.RxNormRelationships{RxCUI: rxcui}

	for _, group := range response.AllRelatedGroup.ConceptGroup {
		for _, prop := range group.ConceptProperties {
			concept := datasources.RxNormConcept{
				RxCUI: prop.RxCUI,
				Name:  prop.Name,
				TTY:   prop.TTY,
			}

			switch prop.TTY {
			case "IN", "MIN":
				rel.Ingredients = append(rel.Ingredients, concept)
			case "BN":
				rel.BrandNames = append(rel.BrandNames, concept)
			case "DF":
				rel.DoseForms = append(rel.DoseForms, concept)
			case "SCDC", "SBDC":
				rel.Components = append(rel.Components, concept)
			default:
				rel.RelatedDrugs = append(rel.RelatedDrugs, concept)
			}
		}
	}

	return rel, nil
}

// GetIngredients returns the active ingredients for a drug product
func (c *Client) GetIngredients(ctx context.Context, rxcui string) ([]datasources.RxNormConcept, error) {
	path := fmt.Sprintf("/rxcui/%s/related.json?tty=IN+MIN", rxcui)

	var response struct {
		RelatedGroup struct {
			ConceptGroup []struct {
				ConceptProperties []struct {
					RxCUI string `json:"rxcui"`
					Name  string `json:"name"`
					TTY   string `json:"tty"`
				} `json:"conceptProperties"`
			} `json:"conceptGroup"`
		} `json:"relatedGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	var concepts []datasources.RxNormConcept
	for _, group := range response.RelatedGroup.ConceptGroup {
		for _, prop := range group.ConceptProperties {
			concepts = append(concepts, datasources.RxNormConcept{
				RxCUI: prop.RxCUI,
				Name:  prop.Name,
				TTY:   prop.TTY,
			})
		}
	}

	return concepts, nil
}

// =============================================================================
// SPL SETID OPERATIONS (Phase 3 - DailyMed Label Fetching)
// =============================================================================

// GetSPLSetID retrieves the SPL SetID for a drug (used for DailyMed label lookup)
// This is the KEY FUNCTION for Phase 3: RxCUI → SPL SetID → DailyMed XML
func (c *Client) GetSPLSetID(ctx context.Context, rxcui string) (string, error) {
	path := fmt.Sprintf("/rxcui/%s/property.json?propName=SPL_SET_ID", rxcui)

	var response struct {
		PropConceptGroup struct {
			PropConcept []struct {
				PropName  string `json:"propName"`
				PropValue string `json:"propValue"`
			} `json:"propConcept"`
		} `json:"propConceptGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return "", err
	}

	for _, prop := range response.PropConceptGroup.PropConcept {
		if prop.PropName == "SPL_SET_ID" {
			return prop.PropValue, nil
		}
	}

	return "", fmt.Errorf("SPL_SET_ID not found for RxCUI: %s", rxcui)
}

// GetSPLSetIDFromNDC is a convenience method: NDC → RxCUI → SPL SetID
// Returns the DailyMed SetID for fetching FDA drug labels
func (c *Client) GetSPLSetIDFromNDC(ctx context.Context, ndc string) (string, error) {
	// Step 1: NDC → RxCUI
	rxcui, err := c.GetRxCUIByNDC(ctx, ndc)
	if err != nil {
		return "", fmt.Errorf("NDC to RxCUI: %w", err)
	}

	// Step 2: Try direct SPL lookup
	setID, err := c.GetSPLSetID(ctx, rxcui)
	if err == nil {
		return setID, nil
	}

	// Step 3: Try related SBD/SCD concepts (Semantic Branded/Clinical Drug)
	// These TTY codes are RxNorm term types that typically have SPL links
	for _, tty := range []datasources.RxNormRelationType{"SBD", "SCD", "GPCK", "BPCK"} {
		related, relErr := c.GetRelatedByType(ctx, rxcui, tty)
		if relErr == nil {
			for _, concept := range related {
				if splID, splErr := c.GetSPLSetID(ctx, concept.RxCUI); splErr == nil {
					return splID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("SPL_SET_ID not found for NDC %s (RxCUI: %s)", ndc, rxcui)
}

// GetSPLSetIDFromDrugName is a convenience method: Drug Name → RxCUI → SPL SetID
func (c *Client) GetSPLSetIDFromDrugName(ctx context.Context, drugName string) (string, error) {
	// Step 1: Name → RxCUI
	rxcui, err := c.GetRxCUIByName(ctx, drugName)
	if err != nil {
		return "", fmt.Errorf("drug name to RxCUI: %w", err)
	}

	// Step 2: RxCUI → SPL SetID
	return c.GetSPLSetID(ctx, rxcui)
}

// GetDrugProperties retrieves all properties for a drug including SPL links
func (c *Client) GetDrugProperties(ctx context.Context, rxcui string) (map[string]string, error) {
	path := fmt.Sprintf("/rxcui/%s/allProperties.json?prop=all", rxcui)

	var response struct {
		PropConceptGroup struct {
			PropConcept []struct {
				PropName  string `json:"propName"`
				PropValue string `json:"propValue"`
			} `json:"propConcept"`
		} `json:"propConceptGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	props := make(map[string]string)
	for _, prop := range response.PropConceptGroup.PropConcept {
		props[prop.PropName] = prop.PropValue
	}

	return props, nil
}

// =============================================================================
// NDC OPERATIONS
// =============================================================================

// GetNDCsByRxCUI returns all NDCs associated with an RxCUI
func (c *Client) GetNDCsByRxCUI(ctx context.Context, rxcui string) ([]string, error) {
	path := fmt.Sprintf("/rxcui/%s/ndcs.json", rxcui)

	var response struct {
		NdcGroup struct {
			NDCList struct {
				NDC []string `json:"ndc"`
			} `json:"ndcList"`
		} `json:"ndcGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	return response.NdcGroup.NDCList.NDC, nil
}

// GetNDCProperties retrieves properties for a specific NDC
func (c *Client) GetNDCProperties(ctx context.Context, ndc string) (*datasources.NDCProperties, error) {
	path := fmt.Sprintf("/ndcproperties.json?id=%s", url.QueryEscape(ndc))

	var response struct {
		NdcPropertyList struct {
			NdcProperty []struct {
				NdcItem      string `json:"ndcItem"`
				RxCUI        string `json:"rxcui"`
				PackagingNdc string `json:"packagingNdc"`
				Status       string `json:"status"`
			} `json:"ndcProperty"`
		} `json:"ndcPropertyList"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	if len(response.NdcPropertyList.NdcProperty) == 0 {
		return nil, fmt.Errorf("no properties found for NDC: %s", ndc)
	}

	prop := response.NdcPropertyList.NdcProperty[0]
	return &datasources.NDCProperties{
		NDC:          prop.NdcItem,
		RxCUI:        prop.RxCUI,
		PackagingNDC: prop.PackagingNdc,
		Status:       prop.Status,
	}, nil
}

// =============================================================================
// INTERACTION OPERATIONS
// =============================================================================

// GetInteractions retrieves drug-drug interactions for an RxCUI
func (c *Client) GetInteractions(ctx context.Context, rxcui string) ([]datasources.DrugInteraction, error) {
	path := fmt.Sprintf("/interaction/interaction.json?rxcui=%s", rxcui)

	var response struct {
		InteractionTypeGroup []struct {
			InteractionType []struct {
				InteractionPair []struct {
					InteractionConcept []struct {
						MinConceptItem struct {
							RxCUI string `json:"rxcui"`
							Name  string `json:"name"`
						} `json:"minConceptItem"`
					} `json:"interactionConcept"`
					Severity    string `json:"severity"`
					Description string `json:"description"`
				} `json:"interactionPair"`
			} `json:"interactionType"`
			SourceName string `json:"sourceName"`
		} `json:"interactionTypeGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	var interactions []datasources.DrugInteraction
	for _, group := range response.InteractionTypeGroup {
		for _, itype := range group.InteractionType {
			for _, pair := range itype.InteractionPair {
				if len(pair.InteractionConcept) >= 2 {
					interactions = append(interactions, datasources.DrugInteraction{
						Drug1RxCUI:  pair.InteractionConcept[0].MinConceptItem.RxCUI,
						Drug1Name:   pair.InteractionConcept[0].MinConceptItem.Name,
						Drug2RxCUI:  pair.InteractionConcept[1].MinConceptItem.RxCUI,
						Drug2Name:   pair.InteractionConcept[1].MinConceptItem.Name,
						Severity:    pair.Severity,
						Description: pair.Description,
						Source:      group.SourceName,
					})
				}
			}
		}
	}

	return interactions, nil
}

// GetInteractionsBetween checks for interactions between multiple drugs
func (c *Client) GetInteractionsBetween(ctx context.Context, rxcuis []string) ([]datasources.DrugInteraction, error) {
	path := fmt.Sprintf("/interaction/list.json?rxcuis=%s", strings.Join(rxcuis, "+"))

	var response struct {
		FullInteractionTypeGroup []struct {
			FullInteractionType []struct {
				InteractionPair []struct {
					InteractionConcept []struct {
						MinConceptItem struct {
							RxCUI string `json:"rxcui"`
							Name  string `json:"name"`
						} `json:"minConceptItem"`
					} `json:"interactionConcept"`
					Severity    string `json:"severity"`
					Description string `json:"description"`
				} `json:"interactionPair"`
			} `json:"fullInteractionType"`
			SourceName string `json:"sourceName"`
		} `json:"fullInteractionTypeGroup"`
	}

	if err := c.doRequest(ctx, path, &response); err != nil {
		return nil, err
	}

	var interactions []datasources.DrugInteraction
	for _, group := range response.FullInteractionTypeGroup {
		for _, itype := range group.FullInteractionType {
			for _, pair := range itype.InteractionPair {
				if len(pair.InteractionConcept) >= 2 {
					interactions = append(interactions, datasources.DrugInteraction{
						Drug1RxCUI:  pair.InteractionConcept[0].MinConceptItem.RxCUI,
						Drug1Name:   pair.InteractionConcept[0].MinConceptItem.Name,
						Drug2RxCUI:  pair.InteractionConcept[1].MinConceptItem.RxCUI,
						Drug2Name:   pair.InteractionConcept[1].MinConceptItem.Name,
						Severity:    pair.Severity,
						Description: pair.Description,
						Source:      group.SourceName,
					})
				}
			}
		}
	}

	return interactions, nil
}

// =============================================================================
// BATCH OPERATIONS
// =============================================================================

// BatchGetDrugs retrieves multiple drugs by RxCUI
func (c *Client) BatchGetDrugs(ctx context.Context, rxcuis []string) (map[string]*datasources.RxNormDrug, error) {
	result := make(map[string]*datasources.RxNormDrug)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(rxcuis))

	// Process in batches to avoid overwhelming the API
	batchSize := 10
	for i := 0; i < len(rxcuis); i += batchSize {
		end := i + batchSize
		if end > len(rxcuis) {
			end = len(rxcuis)
		}

		batch := rxcuis[i:end]
		for _, rxcui := range batch {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()

				drug, err := c.GetDrugByRxCUI(ctx, id)
				if err != nil {
					errChan <- fmt.Errorf("failed to get drug %s: %w", id, err)
					return
				}

				mu.Lock()
				result[id] = drug
				mu.Unlock()
			}(rxcui)
		}

		wg.Wait()
	}

	close(errChan)

	// Log any errors but don't fail the whole batch
	for err := range errChan {
		c.log.WithError(err).Warn("Batch drug fetch error")
	}

	return result, nil
}

// =============================================================================
// HTTP REQUEST HANDLING
// =============================================================================

func (c *Client) doRequest(ctx context.Context, path string, result interface{}) error {
	// Check cache first
	cacheKey := "rxnav:" + path
	if c.cache != nil {
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			return json.Unmarshal(cached, result)
		}
	}

	// Rate limiting
	if c.rateLimiter != nil {
		select {
		case <-c.rateLimiter:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Build request URL
	reqURL := c.config.BaseURL + path

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.config.RetryDelay)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.log.WithError(err).WithField("attempt", attempt+1).Debug("Request failed, retrying")
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("RxNav API error: %d - %s", resp.StatusCode, string(body))
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		// Cache successful response
		if c.cache != nil {
			_ = c.cache.Set(ctx, cacheKey, body, c.config.CacheTTL)
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// refillRateLimiter replenishes the rate limiter periodically
func (c *Client) refillRateLimiter() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Refill tokens
		for i := 0; i < c.config.RateLimitPerSecond; i++ {
			select {
			case c.rateLimiter <- struct{}{}:
			default:
				// Channel full, skip
			}
		}
	}
}
