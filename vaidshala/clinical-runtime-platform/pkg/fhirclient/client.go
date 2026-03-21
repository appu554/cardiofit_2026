package fhirclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
)

// Client communicates with Google Healthcare FHIR Store.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// New creates a FHIR client with OAuth2 service account auth.
func New(cfg GoogleFHIRConfig, logger *zap.Logger) (*Client, error) {
	ctx := context.Background()
	scopes := []string{"https://www.googleapis.com/auth/cloud-healthcare"}

	var httpClient *http.Client
	if cfg.CredentialsPath != "" {
		creds, err := google.FindDefaultCredentials(ctx, scopes...)
		if err != nil {
			return nil, fmt.Errorf("find google credentials: %w", err)
		}
		httpClient = oauth2Transport(creds)
	} else {
		client, err := google.DefaultClient(ctx, scopes...)
		if err != nil {
			return nil, fmt.Errorf("default google client: %w", err)
		}
		httpClient = client
	}
	httpClient.Timeout = 30 * time.Second

	return &Client{
		baseURL:    cfg.BaseURL(),
		httpClient: httpClient,
		logger:     logger,
	}, nil
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
	url := fmt.Sprintf("%s/%s?", c.baseURL, resourceType)
	for k, v := range params {
		url += k + "=" + v + "&"
	}

	resp, err := doWithRetry(c.httpClient, http.MethodGet, url, nil,
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
