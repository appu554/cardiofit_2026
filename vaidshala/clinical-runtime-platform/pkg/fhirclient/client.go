package fhirclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Client communicates with Google Healthcare FHIR Store.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// gcloudTokenSource obtains OAuth2 tokens via the gcloud CLI.
// It shells out to `gcloud auth print-access-token` and caches the result.
type gcloudTokenSource struct {
	mu    sync.Mutex
	token *oauth2.Token
}

func (g *gcloudTokenSource) Token() (*oauth2.Token, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Return cached token if still valid (with 2-min margin).
	if g.token != nil && g.token.Valid() {
		return g.token, nil
	}

	cmd := exec.Command("gcloud", "auth", "print-access-token")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gcloud auth print-access-token: %w", err)
	}

	accessToken := strings.TrimSpace(string(out))
	if accessToken == "" {
		return nil, fmt.Errorf("gcloud returned empty access token — run 'gcloud auth login' first")
	}

	g.token = &oauth2.Token{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(55 * time.Minute), // gcloud tokens last ~60 min
	}
	return g.token, nil
}

// New creates a FHIR client with proper GCP authentication.
// Auth priority: SA key file → Application Default Credentials → gcloud CLI.
// Each method is validated with a health check before accepting it.
func New(cfg GoogleFHIRConfig, logger *zap.Logger) (*Client, error) {
	ctx := context.Background()
	scopes := []string{"https://www.googleapis.com/auth/cloud-healthcare"}
	baseURL := cfg.BaseURL()

	type authAttempt struct {
		name   string
		client *http.Client
	}

	var candidates []authAttempt

	// 1. Try service account key file if configured and exists.
	if cfg.CredentialsPath != "" {
		if _, err := os.Stat(cfg.CredentialsPath); err == nil {
			os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", cfg.CredentialsPath)
			creds, err := google.FindDefaultCredentials(ctx, scopes...)
			if err == nil {
				candidates = append(candidates, authAttempt{
					name:   "service account key (" + cfg.CredentialsPath + ")",
					client: oauth2Transport(creds),
				})
			} else {
				logger.Warn("SA key file found but unusable", zap.Error(err))
			}
			os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
		}
	}

	// 2. Application Default Credentials.
	{
		os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
		creds, err := google.FindDefaultCredentials(ctx, scopes...)
		if err == nil {
			candidates = append(candidates, authAttempt{
				name:   "Application Default Credentials",
				client: oauth2Transport(creds),
			})
		}
	}

	// 3. gcloud CLI token source.
	if _, err := exec.LookPath("gcloud"); err == nil {
		ts := &gcloudTokenSource{}
		if _, err := ts.Token(); err == nil {
			candidates = append(candidates, authAttempt{
				name: "gcloud CLI",
				client: &http.Client{
					Transport: &oauth2.Transport{
						Source: ts,
						Base:   http.DefaultTransport,
					},
				},
			})
		}
	}

	// Validate each candidate with a real health check against the FHIR Store.
	metadataURL := baseURL + "/metadata"
	for _, cand := range candidates {
		cand.client.Timeout = 10 * time.Second
		resp, err := cand.client.Get(metadataURL)
		if err != nil {
			logger.Warn("FHIR auth probe failed", zap.String("method", cand.name), zap.Error(err))
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			cand.client.Timeout = 30 * time.Second
			logger.Info("FHIR auth: using "+cand.name, zap.Int("status", resp.StatusCode))
			return &Client{
				baseURL:    baseURL,
				httpClient: cand.client,
				logger:     logger,
			}, nil
		}
		logger.Warn("FHIR auth probe rejected", zap.String("method", cand.name), zap.Int("status", resp.StatusCode))
	}

	return nil, fmt.Errorf("no GCP auth method succeeded for FHIR Store at %s", baseURL)
}

// NewWithHTTPClient creates a FHIR client with a custom http.Client (for testing).
func NewWithHTTPClient(baseURL string, httpClient *http.Client, logger *zap.Logger) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

func oauth2Transport(creds *google.Credentials) *http.Client {
	return &http.Client{
		Transport: &oauth2RoundTripper{
			base:  http.DefaultTransport,
			creds: creds,
		},
	}
}

type oauth2RoundTripper struct {
	base  http.RoundTripper
	creds *google.Credentials
}

func (t *oauth2RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.creds.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("get oauth2 token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return t.base.RoundTrip(req)
}

// Create creates a FHIR resource. Returns the created resource JSON.
func (c *Client) Create(resourceType string, body []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, resourceType)
	bodyFactory := func() io.Reader { return bytes.NewReader(body) }

	resp, err := doWithRetry(c.httpClient, http.MethodPost, url, bodyFactory,
		map[string]string{"Content-Type": "application/fhir+json"}, c.logger)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR Create %s failed: %d %s", resourceType, resp.StatusCode, string(data))
	}
	return data, nil
}

// Read retrieves a FHIR resource by type and ID.
func (c *Client) Read(resourceType, id string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, id)

	resp, err := doWithRetry(c.httpClient, http.MethodGet, url, nil,
		map[string]string{"Accept": "application/fhir+json"}, c.logger)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("FHIR %s/%s not found", resourceType, id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR Read %s/%s failed: %d", resourceType, id, resp.StatusCode)
	}
	return data, nil
}

// Update replaces a FHIR resource (PUT).
func (c *Client) Update(resourceType, id string, body []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, id)
	bodyFactory := func() io.Reader { return bytes.NewReader(body) }

	resp, err := doWithRetry(c.httpClient, http.MethodPut, url, bodyFactory,
		map[string]string{"Content-Type": "application/fhir+json"}, c.logger)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR Update %s/%s failed: %d", resourceType, id, resp.StatusCode)
	}
	return data, nil
}

// Search queries FHIR resources with query parameters.
func (c *Client) Search(resourceType string, params map[string]string) (json.RawMessage, error) {
	qv := url.Values{}
	for k, v := range params {
		qv.Set(k, v)
	}
	searchURL := fmt.Sprintf("%s/%s?%s", c.baseURL, resourceType, qv.Encode())

	resp, err := doWithRetry(c.httpClient, http.MethodGet, searchURL, nil,
		map[string]string{"Accept": "application/fhir+json"}, c.logger)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR Search %s failed: %d", resourceType, resp.StatusCode)
	}
	return data, nil
}

// TransactionBundle sends a FHIR Transaction Bundle (POST to base URL).
func (c *Client) TransactionBundle(bundle []byte) ([]byte, error) {
	bodyFactory := func() io.Reader { return bytes.NewReader(bundle) }

	resp, err := doWithRetry(c.httpClient, http.MethodPost, c.baseURL, bodyFactory,
		map[string]string{"Content-Type": "application/fhir+json"}, c.logger)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR Transaction failed: %d %s", resp.StatusCode, string(data))
	}
	return data, nil
}

// HealthCheck verifies the FHIR Store is reachable (GET metadata).
func (c *Client) HealthCheck() error {
	url := c.baseURL + "/metadata"
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("FHIR Store health check: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("FHIR Store returned %d", resp.StatusCode)
	}
	return nil
}
