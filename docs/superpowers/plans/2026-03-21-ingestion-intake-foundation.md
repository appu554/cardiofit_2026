# Ingestion + Intake-Onboarding Foundation Plan (Phase 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish service scaffolding, shared packages, database schemas, and API gateway integration for the Ingestion Service (port 8140) and Intake-Onboarding Service (port 8141) — both compile, boot, serve health endpoints, and are routable through the FastAPI gateway.

**Architecture:** Two new Go services under `vaidshala/clinical-runtime-platform/services/`, each with its own `go.mod` + `replace` directive (matching `medication-advisor-engine` pattern). Shared `pkg/fhirclient` in the root module provides Google FHIR Store access. FastAPI gateway gets 4 new route entries + RBAC updates. PostgreSQL schemas use port 5433 (Docker/KB shared instance).

**Tech Stack:** Go 1.25, Gin, pgx/v5, redis/go-redis/v9, zap, prometheus/client_golang, golang.org/x/oauth2, segmentio/kafka-go

**Spec:** `docs/superpowers/specs/2026-03-21-ingestion-intake-onboarding-design.md`

---

## File Structure

### Shared Package (root module)

| File | Responsibility |
|------|---------------|
| `pkg/fhirclient/config.go` | `GoogleFHIRConfig` struct, `BaseURL()` method |
| `pkg/fhirclient/client.go` | `Client` — CRUD + search against Google Healthcare FHIR REST API |
| `pkg/fhirclient/retry.go` | Exponential backoff on 429/5xx (3 attempts, 1s/2s/4s) |
| `pkg/fhirclient/client_test.go` | Unit tests with httptest server |

### Ingestion Service

| File | Responsibility |
|------|---------------|
| `services/ingestion-service/go.mod` | Module with replace directive to root |
| `services/ingestion-service/cmd/ingestion/main.go` | Entry point — config, DB, Redis, Gin server, graceful shutdown |
| `services/ingestion-service/internal/config/config.go` | `Config` struct loaded from env vars |
| `services/ingestion-service/internal/api/server.go` | `Server` struct with Gin router, DI, middleware |
| `services/ingestion-service/internal/api/routes.go` | Route registration (health, FHIR, ingest, internal, admin) |
| `services/ingestion-service/internal/api/health.go` | `/healthz`, `/readyz`, `/startupz` handlers |
| `services/ingestion-service/internal/canonical/observation.go` | `CanonicalObservation` struct (18 fields) |
| `services/ingestion-service/internal/canonical/flags.go` | Flag constants: `CRITICAL_VALUE`, `IMPLAUSIBLE`, etc. |
| `services/ingestion-service/internal/pipeline/interfaces.go` | 6 pipeline stage interfaces: Receiver, Parser, Normalizer, Validator, Mapper, Router |
| `services/ingestion-service/internal/kafka/envelope.go` | Kafka message envelope struct |
| `services/ingestion-service/migrations/001_init.sql` | `lab_code_mappings`, `dlq_messages`, `patient_pending_queue` |
| `services/ingestion-service/Makefile` | build, run, test, docker, health targets |
| `services/ingestion-service/Dockerfile` | Multi-stage Alpine build |

### Intake-Onboarding Service

| File | Responsibility |
|------|---------------|
| `services/intake-onboarding-service/go.mod` | Module with replace directive to root |
| `services/intake-onboarding-service/cmd/intake/main.go` | Entry point — config, DB, Redis, Gin server, graceful shutdown |
| `services/intake-onboarding-service/internal/config/config.go` | `Config` struct loaded from env vars |
| `services/intake-onboarding-service/internal/api/server.go` | `Server` struct with Gin router, DI, middleware |
| `services/intake-onboarding-service/internal/api/routes.go` | Route registration (health, FHIR CRUD, $operations) |
| `services/intake-onboarding-service/internal/api/health.go` | `/healthz`, `/readyz`, `/startupz` handlers |
| `services/intake-onboarding-service/internal/enrollment/states.go` | 8-state enum + valid transitions map |
| `services/intake-onboarding-service/internal/enrollment/transitions.go` | `StateMachine` with `Transition()` method |
| `services/intake-onboarding-service/internal/kafka/envelope.go` | Kafka message envelope (same struct as ingestion) |
| `services/intake-onboarding-service/migrations/001_init.sql` | `enrollments`, `slot_events`, `current_slots`, `flow_positions`, `review_queue` |
| `services/intake-onboarding-service/Makefile` | build, run, test, docker, health targets |
| `services/intake-onboarding-service/Dockerfile` | Multi-stage Alpine build |

### API Gateway (Python modifications)

| File | Change |
|------|--------|
| `backend/services/api-gateway/app/config.py` | Add `INGESTION_SERVICE_URL`, `INTAKE_SERVICE_URL` |
| `backend/services/api-gateway/app/api/proxy.py` | Replace `device_ingestion` route, add 4 new entries, update `check_permissions_with_auth_service()` |
| `backend/services/api-gateway/app/middleware/rbac.py` | Add intake/ingestion ROUTE_PERMISSIONS + ROLE_ROUTE_RESTRICTIONS |

---

## Task 1: Shared FHIR Client Package (`pkg/fhirclient`)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/pkg/fhirclient/config.go`
- Create: `vaidshala/clinical-runtime-platform/pkg/fhirclient/client.go`
- Create: `vaidshala/clinical-runtime-platform/pkg/fhirclient/retry.go`
- Create: `vaidshala/clinical-runtime-platform/pkg/fhirclient/client_test.go`

**Reference:** KB-20's `internal/fhir/fhir_client.go` uses the same OAuth2 + retry pattern. We're creating a new shared version in `pkg/` so both services can import it.

- [ ] **Step 1: Write config.go**

```go
// vaidshala/clinical-runtime-platform/pkg/fhirclient/config.go
package fhirclient

import "fmt"

// GoogleFHIRConfig holds connection details for Google Healthcare FHIR Store.
type GoogleFHIRConfig struct {
	Enabled         bool   `json:"enabled"`
	ProjectID       string `json:"project_id"`
	Location        string `json:"location"`
	DatasetID       string `json:"dataset_id"`
	FhirStoreID     string `json:"fhir_store_id"`
	CredentialsPath string `json:"credentials_path"`
}

// BaseURL returns the FHIR Store REST base URL.
func (c GoogleFHIRConfig) BaseURL() string {
	return fmt.Sprintf(
		"https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
		c.ProjectID, c.Location, c.DatasetID, c.FhirStoreID,
	)
}
```

- [ ] **Step 2: Write retry.go**

```go
// vaidshala/clinical-runtime-platform/pkg/fhirclient/retry.go
package fhirclient

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var retryDelays = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

// doWithRetry executes an HTTP request with exponential backoff on 429/5xx.
// The bodyFactory is called on each attempt to produce a fresh request body.
func doWithRetry(
	client *http.Client,
	method, url string,
	bodyFactory func() io.Reader,
	headers map[string]string,
	logger *zap.Logger,
) (*http.Response, error) {
	var lastErr error
	for attempt, delay := range retryDelays {
		var body io.Reader
		if bodyFactory != nil {
			body = bodyFactory()
		}

		req, err := http.NewRequest(method, url, body)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			logger.Warn("FHIR request failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Error(err),
			)
			time.Sleep(delay)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = fmt.Errorf("FHIR Store returned %d", resp.StatusCode)
			logger.Warn("FHIR Store returned retryable status",
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt+1),
			)
			time.Sleep(delay)
			continue
		}

		return resp, nil
	}
	return nil, fmt.Errorf("all %d retries exhausted: %w", len(retryDelays), lastErr)
}
```

- [ ] **Step 3: Write client.go**

```go
// vaidshala/clinical-runtime-platform/pkg/fhirclient/client.go
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

// New creates a FHIR client. If cfg.CredentialsPath is set, it uses OAuth2
// service account auth; otherwise it falls back to Application Default Credentials.
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
```

- [ ] **Step 4: Write client_test.go**

```go
// vaidshala/clinical-runtime-platform/pkg/fhirclient/client_test.go
package fhirclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestGoogleFHIRConfig_BaseURL(t *testing.T) {
	cfg := GoogleFHIRConfig{
		ProjectID:   "cardiofit-905a8",
		Location:    "asia-south1",
		DatasetID:   "clinical-synthesis-hub",
		FhirStoreID: "fhir-store",
	}
	want := "https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store/fhir"
	if got := cfg.BaseURL(); got != want {
		t.Errorf("BaseURL() = %q, want %q", got, want)
	}
}

func TestClient_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/Patient" {
			t.Errorf("expected /Patient, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"resourceType":"Patient","id":"123"}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Create("Patient", []byte(`{"resourceType":"Patient"}`))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if string(data) != `{"resourceType":"Patient","id":"123"}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_Read(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/123" {
			t.Errorf("expected /Patient/123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resourceType":"Patient","id":"123"}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Read("Patient", "123")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != `{"resourceType":"Patient","id":"123"}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_Read_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	_, err := client.Read("Patient", "999")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClient_RetryOn429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resourceType":"Patient","id":"123"}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Read("Patient", "123")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if string(data) != `{"resourceType":"Patient","id":"123"}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_Search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("patient") != "123" {
			t.Errorf("expected patient=123 param")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resourceType":"Bundle","total":1}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Search("Observation", map[string]string{"patient": "123"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if string(data) != `{"resourceType":"Bundle","total":1}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata" {
			t.Errorf("expected /metadata, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	if err := client.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}
```

- [ ] **Step 5: Add dependencies to root go.mod**

The `pkg/fhirclient` package imports `go.uber.org/zap` and `golang.org/x/oauth2/google`, which must be in the root module's `go.mod` (since `pkg/` is part of the root module). The current root `go.mod` only has `gin`, `uuid`, and `yaml.v3`.

Run: `cd vaidshala/clinical-runtime-platform && go get go.uber.org/zap golang.org/x/oauth2`
Expected: Root `go.mod` and `go.sum` updated with zap + oauth2 + transitive deps.

Then commit:
```bash
git add vaidshala/clinical-runtime-platform/go.mod vaidshala/clinical-runtime-platform/go.sum
git commit -m "deps: add zap and oauth2 to root module for pkg/fhirclient"
```

- [ ] **Step 6: Run tests**

Run: `cd vaidshala/clinical-runtime-platform && go test ./pkg/fhirclient/... -v -count=1`
Expected: All 6 tests PASS (config URL, create, read, read not found, retry on 429, search, health check)

- [ ] **Step 7: Commit**

```bash
git add vaidshala/clinical-runtime-platform/pkg/fhirclient/
git commit -m "feat(fhirclient): add shared Google FHIR Store client package

OAuth2 service account auth, exponential backoff retry on 429/5xx,
CRUD + Search + TransactionBundle operations. Inspired by KB-20's
fhir_client.go pattern but in pkg/ for cross-service import."
```

---

## Task 2: Ingestion Service Scaffolding

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/go.mod`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/cmd/ingestion/main.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/config/config.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/server.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/routes.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/health.go`

**Reference:** Follows KB-20 pattern (zap + Gin + dependency injection) since these are infrastructure-tier services.

- [ ] **Step 1: Write go.mod**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/go.mod
module github.com/cardiofit/ingestion-service

go 1.25.1

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/prometheus/client_golang v1.20.5
	github.com/redis/go-redis/v9 v9.7.3
	github.com/segmentio/kafka-go v0.4.47
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.27.0
	golang.org/x/oauth2 v0.25.0
	vaidshala/clinical-runtime-platform v0.0.0
)

// Use local parent module for pkg/fhirclient
replace vaidshala/clinical-runtime-platform => ../../
```

- [ ] **Step 2: Write config.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/config/config.go
package config

import (
	"os"
	"strconv"
	"time"

	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	FHIR     fhirclient.GoogleFHIRConfig
	Kafka    KafkaConfig

	Environment string
	LogLevel            string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL             string
	MaxConnections  int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8140"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://ingestion_user:ingestion_password@localhost:5433/ingestion_service?sslmode=disable"),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6380"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 2),
		},
		FHIR: fhirclient.GoogleFHIRConfig{
			Enabled:         getEnvAsBool("FHIR_ENABLED", true),
			ProjectID:       getEnv("FHIR_PROJECT_ID", "cardiofit-905a8"),
			Location:        getEnv("FHIR_LOCATION", "asia-south1"),
			DatasetID:       getEnv("FHIR_DATASET_ID", "clinical-synthesis-hub"),
			FhirStoreID:     getEnv("FHIR_STORE_ID", "fhir-store"),
			CredentialsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "ingestion-service"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}, nil
}

func (c *Config) IsDevelopment() bool { return c.Environment == "development" }

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
```

- [ ] **Step 3: Write health.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/health.go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ingestion-service"})
}

func (s *Server) handleReadyz(c *gin.Context) {
	checks := gin.H{}
	allReady := true

	// Check PostgreSQL
	if s.db != nil {
		if err := s.db.Ping(c.Request.Context()); err != nil {
			checks["postgresql"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["postgresql"] = "ok"
		}
	}

	// Check Redis
	if s.redis != nil {
		if err := s.redis.Ping(c.Request.Context()).Err(); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["redis"] = "ok"
		}
	}

	// Check FHIR Store
	if s.fhirClient != nil {
		if err := s.fhirClient.HealthCheck(); err != nil {
			checks["fhir_store"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["fhir_store"] = "ok"
		}
	}

	status := http.StatusOK
	if !allReady {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"ready": allReady, "checks": checks})
}

func (s *Server) handleStartupz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"started": true, "service": "ingestion-service"})
}
```

- [ ] **Step 4: Write server.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/server.go
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

type Server struct {
	Router     *gin.Engine
	config     *config.Config
	db         *pgxpool.Pool
	redis      *redis.Client
	fhirClient *fhirclient.Client
	logger     *zap.Logger
}

func NewServer(
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	fhirClient *fhirclient.Client,
	logger *zap.Logger,
) *Server {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		Router:     router,
		config:     cfg,
		db:         db,
		redis:      redisClient,
		fhirClient: fhirClient,
		logger:     logger,
	}

	router.Use(s.metricsMiddleware())
	router.Use(s.corsMiddleware())
	s.setupRoutes()

	return s
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		_ = duration // TODO: wire to prometheus histogram in Phase 2
		_ = strconv.Itoa(c.Writer.Status())
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Role")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
```

- [ ] **Step 5: Write routes.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/api/routes.go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	// Infrastructure
	s.Router.GET("/healthz", s.handleHealthz)
	s.Router.GET("/readyz", s.handleReadyz)
	s.Router.GET("/startupz", s.handleStartupz)
	s.Router.GET("/metrics", s.prometheusHandler())

	// FHIR-compliant inbound (Phase 2)
	fhir := s.Router.Group("/fhir")
	{
		fhir.POST("", s.stubHandler("FHIR Transaction Bundle"))
		fhir.POST("/Observation", s.stubHandler("FHIR Observation"))
		fhir.POST("/DiagnosticReport", s.stubHandler("FHIR DiagnosticReport"))
		fhir.POST("/MedicationStatement", s.stubHandler("FHIR MedicationStatement"))
	}

	// Source-specific receivers (Phase 2-4)
	ingest := s.Router.Group("/ingest")
	{
		ingest.POST("/ehr/hl7v2", s.stubHandler("HL7v2 ingest"))
		ingest.POST("/ehr/fhir", s.stubHandler("FHIR passthrough"))
		ingest.POST("/labs/:labId", s.stubHandler("Lab ingest"))
		ingest.POST("/devices", s.stubHandler("Device ingest"))
		ingest.POST("/wearables/:provider", s.stubHandler("Wearable ingest"))
		ingest.POST("/abdm/data-push", s.stubHandler("ABDM data push"))
	}

	// Internal (service-to-service)
	internal := s.Router.Group("/internal")
	{
		internal.POST("/hpi", s.stubHandler("HPI slot data from Intake"))
	}

	// Admin/Dashboard
	s.Router.GET("/$source-status", s.stubHandler("Source status"))
}

// stubHandler returns a 501 Not Implemented response with the endpoint name.
// These stubs are replaced with real handlers in Phase 2+.
func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in Phase 2",
		})
	}
}
```

- [ ] **Step 6: Write main.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/cmd/ingestion/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/api"
	"github.com/cardiofit/ingestion-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting Ingestion Service...")

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Connect PostgreSQL
	dbPool, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer dbPool.Close()
	logger.Info("Connected to PostgreSQL")

	// Connect Redis
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}
	opt.Password = cfg.Redis.Password
	opt.DB = cfg.Redis.DB
	redisClient := redis.NewClient(opt)
	defer redisClient.Close()
	logger.Info("Connected to Redis")

	// Create FHIR client (optional — disabled in dev if no credentials)
	var fhirClient *fhirclient.Client
	if cfg.FHIR.Enabled {
		fhirClient, err = fhirclient.New(cfg.FHIR, logger)
		if err != nil {
			logger.Warn("FHIR Store client disabled — no credentials", zap.Error(err))
		} else {
			logger.Info("FHIR Store client initialized")
		}
	}

	// Create HTTP server
	server := api.NewServer(cfg, dbPool, redisClient, fhirClient, logger)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: server.Router,
	}

	go func() {
		logger.Info("Ingestion Service listening", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Ingestion Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Ingestion Service stopped")
}
```

- [ ] **Step 7: Run go mod tidy and verify compilation**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go mod tidy && go build ./cmd/ingestion/`
Expected: Binary compiles without errors

- [ ] **Step 8: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/
git commit -m "feat(ingestion): scaffold service with Gin server, config, health endpoints

Port 8140. Per-service go.mod with replace directive. Stub handlers for
all FHIR and ingest endpoints (return 501 until Phase 2). Health checks
for PostgreSQL, Redis, and FHIR Store."
```

---

## Task 3: Intake-Onboarding Service Scaffolding

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/go.mod`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/cmd/intake/main.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/config/config.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/server.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/health.go`

**Reference:** Same pattern as Task 2 but with intake-specific routes and port 8141.

- [ ] **Step 1: Write go.mod**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/go.mod
module github.com/cardiofit/intake-onboarding-service

go 1.25.1

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/prometheus/client_golang v1.20.5
	github.com/redis/go-redis/v9 v9.7.3
	github.com/segmentio/kafka-go v0.4.47
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.27.0
	golang.org/x/oauth2 v0.25.0
	vaidshala/clinical-runtime-platform v0.0.0
)

// Use local parent module for pkg/fhirclient
replace vaidshala/clinical-runtime-platform => ../../
```

- [ ] **Step 2: Write config.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/config/config.go
package config

import (
	"os"
	"strconv"
	"time"

	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	FHIR     fhirclient.GoogleFHIRConfig
	Kafka    KafkaConfig

	IngestionServiceURL string // For HPI forwarding to POST /internal/hpi
	Environment         string
	LogLevel            string
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL             string
	MaxConnections  int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8141"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://intake_user:intake_password@localhost:5433/intake_service?sslmode=disable"),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6380"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 3),
		},
		FHIR: fhirclient.GoogleFHIRConfig{
			Enabled:         getEnvAsBool("FHIR_ENABLED", true),
			ProjectID:       getEnv("FHIR_PROJECT_ID", "cardiofit-905a8"),
			Location:        getEnv("FHIR_LOCATION", "asia-south1"),
			DatasetID:       getEnv("FHIR_DATASET_ID", "clinical-synthesis-hub"),
			FhirStoreID:     getEnv("FHIR_STORE_ID", "fhir-store"),
			CredentialsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "intake-onboarding-service"),
		},
		IngestionServiceURL: getEnv("INGESTION_SERVICE_URL", "http://localhost:8140"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
	}, nil
}

func (c *Config) IsDevelopment() bool { return c.Environment == "development" }

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
```

- [ ] **Step 3: Write health.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/health.go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "intake-onboarding-service"})
}

func (s *Server) handleReadyz(c *gin.Context) {
	checks := gin.H{}
	allReady := true

	if s.db != nil {
		if err := s.db.Ping(c.Request.Context()); err != nil {
			checks["postgresql"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["postgresql"] = "ok"
		}
	}

	if s.redis != nil {
		if err := s.redis.Ping(c.Request.Context()).Err(); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["redis"] = "ok"
		}
	}

	if s.fhirClient != nil {
		if err := s.fhirClient.HealthCheck(); err != nil {
			checks["fhir_store"] = "unhealthy: " + err.Error()
			allReady = false
		} else {
			checks["fhir_store"] = "ok"
		}
	}

	status := http.StatusOK
	if !allReady {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"ready": allReady, "checks": checks})
}

func (s *Server) handleStartupz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"started": true, "service": "intake-onboarding-service"})
}
```

- [ ] **Step 4: Write server.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/server.go
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

type Server struct {
	Router     *gin.Engine
	config     *config.Config
	db         *pgxpool.Pool
	redis      *redis.Client
	fhirClient *fhirclient.Client
	logger     *zap.Logger
}

func NewServer(
	cfg *config.Config,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	fhirClient *fhirclient.Client,
	logger *zap.Logger,
) *Server {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	s := &Server{
		Router:     router,
		config:     cfg,
		db:         db,
		redis:      redisClient,
		fhirClient: fhirClient,
		logger:     logger,
	}

	router.Use(s.metricsMiddleware())
	router.Use(s.corsMiddleware())
	s.setupRoutes()

	return s
}

func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		_ = duration
		_ = strconv.Itoa(c.Writer.Status())
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Role")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
```

- [ ] **Step 5: Write routes.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) setupRoutes() {
	// Infrastructure
	s.Router.GET("/healthz", s.handleHealthz)
	s.Router.GET("/readyz", s.handleReadyz)
	s.Router.GET("/startupz", s.handleStartupz)
	s.Router.GET("/metrics", s.prometheusHandler())

	// FHIR CRUD (Phase 3)
	fhir := s.Router.Group("/fhir")
	{
		// Patient
		fhir.POST("/Patient", s.stubHandler("Create Patient"))
		fhir.GET("/Patient/:id", s.stubHandler("Read Patient"))
		fhir.PUT("/Patient/:id", s.stubHandler("Update Patient"))
		fhir.GET("/Patient", s.stubHandler("Search Patient"))

		// Observation
		fhir.POST("/Observation", s.stubHandler("Create Observation"))
		fhir.GET("/Observation", s.stubHandler("Search Observation"))

		// Encounter
		fhir.POST("/Encounter", s.stubHandler("Create Encounter"))
		fhir.PUT("/Encounter/:id", s.stubHandler("Update Encounter"))
		fhir.GET("/Encounter/:id", s.stubHandler("Read Encounter"))

		// Other resources
		fhir.POST("/MedicationStatement", s.stubHandler("Create MedicationStatement"))
		fhir.GET("/MedicationStatement", s.stubHandler("Search MedicationStatement"))
		fhir.GET("/DetectedIssue", s.stubHandler("Search DetectedIssue"))
		fhir.POST("/Condition", s.stubHandler("Create Condition"))
		fhir.GET("/Condition", s.stubHandler("Search Condition"))
		fhir.POST("", s.stubHandler("FHIR Transaction Bundle"))

		// Custom $operations — Enrollment
		fhir.POST("/Patient/$enroll", s.stubHandler("Enroll Patient"))
		fhir.POST("/Patient/:id/$verify-otp", s.stubHandler("Verify OTP"))
		fhir.POST("/Patient/:id/$link-abha", s.stubHandler("Link ABHA"))

		// Custom $operations — Safety
		fhir.POST("/Patient/:id/$evaluate-safety", s.stubHandler("Evaluate Safety"))
		fhir.POST("/Encounter/:id/$fill-slot", s.stubHandler("Fill Slot"))

		// Custom $operations — Review
		fhir.POST("/Encounter/:id/$submit-review", s.stubHandler("Submit Review"))
		fhir.POST("/Encounter/:id/$approve", s.stubHandler("Approve"))
		fhir.POST("/Encounter/:id/$request-clarification", s.stubHandler("Request Clarification"))
		fhir.POST("/Encounter/:id/$escalate", s.stubHandler("Escalate"))

		// Custom $operations — Check-in
		fhir.POST("/Patient/:id/$checkin", s.stubHandler("Start Checkin"))
		fhir.POST("/Encounter/:id/$checkin-slot", s.stubHandler("Fill Checkin Slot"))

		// Custom $operations — Co-enrollee
		fhir.POST("/Patient/:id/$register-co-enrollee", s.stubHandler("Register Co-enrollee"))
	}
}

func (s *Server) stubHandler(name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"status":   "not_implemented",
			"endpoint": name,
			"message":  "This endpoint will be implemented in Phase 3",
		})
	}
}
```

- [ ] **Step 6: Write main.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/cmd/intake/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/cardiofit/intake-onboarding-service/internal/api"
	"github.com/cardiofit/intake-onboarding-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting Intake-Onboarding Service...")

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Connect PostgreSQL
	dbPool, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer dbPool.Close()
	logger.Info("Connected to PostgreSQL")

	// Connect Redis
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}
	opt.Password = cfg.Redis.Password
	opt.DB = cfg.Redis.DB
	redisClient := redis.NewClient(opt)
	defer redisClient.Close()
	logger.Info("Connected to Redis")

	// Create FHIR client (optional — disabled in dev if no credentials)
	var fhirClient *fhirclient.Client
	if cfg.FHIR.Enabled {
		fhirClient, err = fhirclient.New(cfg.FHIR, logger)
		if err != nil {
			logger.Warn("FHIR Store client disabled — no credentials", zap.Error(err))
		} else {
			logger.Info("FHIR Store client initialized")
		}
	}

	// Create HTTP server
	server := api.NewServer(cfg, dbPool, redisClient, fhirClient, logger)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Server.Port),
		Handler: server.Router,
	}

	go func() {
		logger.Info("Intake-Onboarding Service listening", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Intake-Onboarding Service...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Intake-Onboarding Service stopped")
}
```

- [ ] **Step 7: Run go mod tidy and verify compilation**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go mod tidy && go build ./cmd/intake/`
Expected: Binary compiles without errors

- [ ] **Step 8: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/
git commit -m "feat(intake): scaffold service with Gin server, config, health endpoints

Port 8141. All FHIR CRUD + 12 custom \$operations registered as stubs
(return 501 until Phase 3). Health checks for PostgreSQL, Redis, FHIR Store."
```

---

## Task 4: Ingestion Canonical Types + Pipeline Interfaces

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/observation.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/flags.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/interfaces.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/envelope.go`
- Test: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/observation_test.go`

- [ ] **Step 1: Write canonical/observation.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/observation.go
package canonical

import (
	"time"

	"github.com/google/uuid"
)

// CanonicalObservation is the normalized intermediate form for all ingested
// health data. Every adapter converts its native format into this struct
// before the pipeline stages process it.
type CanonicalObservation struct {
	ID              uuid.UUID          `json:"id"`
	PatientID       uuid.UUID          `json:"patient_id"`
	TenantID        uuid.UUID          `json:"tenant_id"`
	SourceType      SourceType         `json:"source_type"`
	SourceID        string             `json:"source_id"`
	ObservationType ObservationType    `json:"observation_type"`
	LOINCCode       string             `json:"loinc_code"`
	SNOMEDCode      string             `json:"snomed_code,omitempty"`
	Value           float64            `json:"value"`
	ValueString     string             `json:"value_string,omitempty"`
	Unit            string             `json:"unit"`
	Timestamp       time.Time          `json:"timestamp"`
	QualityScore    float64            `json:"quality_score"`
	Flags           []Flag             `json:"flags,omitempty"`
	DeviceContext   *DeviceContext     `json:"device_context,omitempty"`
	ClinicalContext *ClinicalContext   `json:"clinical_context,omitempty"`
	ABDMContext     *ABDMContext       `json:"abdm_context,omitempty"`
	RawPayload      []byte             `json:"raw_payload,omitempty"`
}

// SourceType identifies where the data came from.
type SourceType string

const (
	SourceEHR             SourceType = "EHR"
	SourceABDM            SourceType = "ABDM"
	SourceLab             SourceType = "LAB"
	SourcePatientReported SourceType = "PATIENT_REPORTED"
	SourceHPI             SourceType = "HPI"
	SourceDevice          SourceType = "DEVICE"
	SourceWearable        SourceType = "WEARABLE"
)

// ObservationType categorizes the observation for Kafka topic routing.
type ObservationType string

const (
	ObsVitals          ObservationType = "VITALS"
	ObsLabs            ObservationType = "LABS"
	ObsMedications     ObservationType = "MEDICATIONS"
	ObsPatientReported ObservationType = "PATIENT_REPORTED"
	ObsHPI             ObservationType = "HPI"
	ObsDeviceData      ObservationType = "DEVICE_DATA"
	ObsABDMRecords     ObservationType = "ABDM_RECORDS"
	ObsGeneral         ObservationType = "GENERAL"
)

// DeviceContext holds device-specific metadata.
type DeviceContext struct {
	DeviceID     string `json:"device_id"`
	DeviceType   string `json:"device_type"`
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	FirmwareVer  string `json:"firmware_version,omitempty"`
}

// ClinicalContext holds clinical metadata.
type ClinicalContext struct {
	EncounterID string `json:"encounter_id,omitempty"`
	OrderID     string `json:"order_id,omitempty"`
	Method      string `json:"method,omitempty"`
	BodySite    string `json:"body_site,omitempty"`
}

// ABDMContext holds ABDM (Ayushman Bharat Digital Mission) metadata.
type ABDMContext struct {
	ConsentID    string `json:"consent_id"`
	HIURequestID string `json:"hiu_request_id"`
	CareContext  string `json:"care_context,omitempty"`
}
```

- [ ] **Step 2: Write canonical/flags.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/flags.go
package canonical

// Flag represents a quality or clinical flag on an observation.
type Flag string

const (
	FlagCriticalValue Flag = "CRITICAL_VALUE"
	FlagImplausible   Flag = "IMPLAUSIBLE"
	FlagLowQuality    Flag = "LOW_QUALITY"
	FlagUnmappedCode  Flag = "UNMAPPED_CODE"
	FlagStale         Flag = "STALE"
	FlagDuplicate     Flag = "DUPLICATE"
	FlagManualEntry   Flag = "MANUAL_ENTRY"
)
```

- [ ] **Step 3: Write pipeline/interfaces.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/interfaces.go
package pipeline

import (
	"context"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// Receiver accepts raw bytes from a source adapter and returns the raw payload.
type Receiver interface {
	Receive(ctx context.Context, raw []byte) ([]byte, error)
}

// Parser converts raw bytes into one or more CanonicalObservation structs.
type Parser interface {
	Parse(ctx context.Context, raw []byte, sourceType canonical.SourceType, sourceID string) ([]canonical.CanonicalObservation, error)
}

// Normalizer applies unit conversion, code mapping, and temporal alignment.
type Normalizer interface {
	Normalize(ctx context.Context, obs *canonical.CanonicalObservation) error
}

// Validator checks clinical ranges, completeness, and quality scoring.
type Validator interface {
	Validate(ctx context.Context, obs *canonical.CanonicalObservation) error
}

// Mapper converts a CanonicalObservation to a FHIR R4 resource (JSON bytes).
type Mapper interface {
	MapToFHIR(ctx context.Context, obs *canonical.CanonicalObservation) ([]byte, error)
}

// Router selects the Kafka topic and partition based on observation category
// and urgency flags.
type Router interface {
	Route(ctx context.Context, obs *canonical.CanonicalObservation) (topic string, partitionKey string, err error)
}
```

- [ ] **Step 4: Write kafka/envelope.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/envelope.go
package kafka

import (
	"time"

	"github.com/google/uuid"
)

// Envelope is the Kafka message wrapper used by both Ingestion and Intake services.
type Envelope struct {
	EventID          uuid.UUID              `json:"eventId"`
	EventType        string                 `json:"eventType"`
	SourceType       string                 `json:"sourceType"`
	PatientID        uuid.UUID              `json:"patientId"`
	TenantID         uuid.UUID              `json:"tenantId"`
	Timestamp        time.Time              `json:"timestamp"`
	FHIRResourceType string                 `json:"fhirResourceType"`
	FHIRResourceID   string                 `json:"fhirResourceId"`
	Payload          map[string]interface{} `json:"payload"`
	QualityScore     float64                `json:"qualityScore,omitempty"`
	Flags            []string               `json:"flags,omitempty"`
	TraceID          string                 `json:"traceId,omitempty"`
}
```

- [ ] **Step 5: Write observation_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/observation_test.go
package canonical

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCanonicalObservation_SourceTypes(t *testing.T) {
	sources := []SourceType{
		SourceEHR, SourceABDM, SourceLab,
		SourcePatientReported, SourceHPI, SourceDevice, SourceWearable,
	}
	if len(sources) != 7 {
		t.Errorf("expected 7 source types, got %d", len(sources))
	}
}

func TestCanonicalObservation_Construct(t *testing.T) {
	obs := CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      SourceLab,
		SourceID:        "thyrocare",
		ObservationType: ObsLabs,
		LOINCCode:       "33914-3",
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
		QualityScore:    0.95,
		Flags:           []Flag{FlagCriticalValue},
	}

	if obs.SourceType != SourceLab {
		t.Errorf("expected LAB source, got %s", obs.SourceType)
	}
	if obs.LOINCCode != "33914-3" {
		t.Errorf("expected LOINC 33914-3, got %s", obs.LOINCCode)
	}
	if len(obs.Flags) != 1 || obs.Flags[0] != FlagCriticalValue {
		t.Errorf("expected CRITICAL_VALUE flag")
	}
}

func TestFlags_Constants(t *testing.T) {
	flags := []Flag{
		FlagCriticalValue, FlagImplausible, FlagLowQuality,
		FlagUnmappedCode, FlagStale, FlagDuplicate, FlagManualEntry,
	}
	if len(flags) != 7 {
		t.Errorf("expected 7 flag constants, got %d", len(flags))
	}
}
```

- [ ] **Step 6: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/canonical/... -v -count=1`
Expected: All 3 tests PASS

- [ ] **Step 7: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/canonical/ \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/pipeline/ \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/
git commit -m "feat(ingestion): add canonical types, pipeline interfaces, Kafka envelope

CanonicalObservation (18 fields), 7 source types, 8 observation types,
7 quality flags. 6 pipeline stage interfaces (Receiver → Router).
Kafka Envelope struct matching spec section 6.3."
```

---

## Task 5: Intake Enrollment State Machine

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/enrollment/states.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/enrollment/transitions.go`
- Test: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/enrollment/states_test.go`

- [ ] **Step 1: Write states.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/enrollment/states.go
package enrollment

import "fmt"

// State represents a step in the 8-state enrollment lifecycle.
type State string

const (
	StateCreated          State = "CREATED"
	StateIdentityVerified State = "IDENTITY_VERIFIED"
	StateIntakeReady      State = "INTAKE_READY"
	StateIntakeInProgress State = "INTAKE_IN_PROGRESS"
	StateHardStopped      State = "HARD_STOPPED"
	StateIntakePaused     State = "INTAKE_PAUSED"
	StateIntakeCompleted  State = "INTAKE_COMPLETED"
	StateEnrolled         State = "ENROLLED"
)

// AllStates returns the 8 enrollment states in lifecycle order.
func AllStates() []State {
	return []State{
		StateCreated, StateIdentityVerified, StateIntakeReady,
		StateIntakeInProgress, StateHardStopped, StateIntakePaused,
		StateIntakeCompleted, StateEnrolled,
	}
}

// validTransitions defines which state transitions are allowed.
var validTransitions = map[State][]State{
	StateCreated:          {StateIdentityVerified},
	StateIdentityVerified: {StateIntakeReady},
	StateIntakeReady:      {StateIntakeInProgress},
	StateIntakeInProgress: {StateHardStopped, StateIntakePaused, StateIntakeCompleted},
	StateHardStopped:      {}, // terminal — requires physician escalation
	StateIntakePaused:     {StateIntakeInProgress}, // resume after timeout/reminder
	StateIntakeCompleted:  {StateEnrolled},
	StateEnrolled:         {}, // terminal
}

// CanTransition checks if moving from `from` to `to` is a valid transition.
func CanTransition(from, to State) bool {
	targets, exists := validTransitions[from]
	if !exists {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// ErrInvalidTransition is returned when a state transition is not allowed.
type ErrInvalidTransition struct {
	From State
	To   State
}

func (e *ErrInvalidTransition) Error() string {
	return fmt.Sprintf("invalid enrollment transition: %s → %s", e.From, e.To)
}
```

- [ ] **Step 2: Write transitions.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/enrollment/transitions.go
package enrollment

import (
	"time"

	"github.com/google/uuid"
)

// ChannelType identifies the enrollment channel.
type ChannelType string

const (
	ChannelCorporate  ChannelType = "CORPORATE"
	ChannelInsurance  ChannelType = "INSURANCE"
	ChannelGovernment ChannelType = "GOVERNMENT"
)

// Enrollment holds the enrollment state for a patient.
type Enrollment struct {
	PatientID          uuid.UUID   `json:"patient_id"`
	TenantID           uuid.UUID   `json:"tenant_id"`
	ChannelType        ChannelType `json:"channel_type"`
	State              State       `json:"state"`
	EncounterID        uuid.UUID   `json:"encounter_id"`
	AssignedPharmacist *uuid.UUID  `json:"assigned_pharmacist,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// Transition attempts to move the enrollment to a new state.
// Returns ErrInvalidTransition if the transition is not allowed.
func (e *Enrollment) Transition(to State) error {
	if !CanTransition(e.State, to) {
		return &ErrInvalidTransition{From: e.State, To: to}
	}
	e.State = to
	e.UpdatedAt = time.Now().UTC()
	return nil
}

// IsTerminal returns true if the enrollment is in a terminal state.
func (e *Enrollment) IsTerminal() bool {
	return e.State == StateHardStopped || e.State == StateEnrolled
}
```

- [ ] **Step 3: Write states_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/enrollment/states_test.go
package enrollment

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAllStates_Count(t *testing.T) {
	states := AllStates()
	if len(states) != 8 {
		t.Errorf("expected 8 states, got %d", len(states))
	}
}

func TestCanTransition_HappyPath(t *testing.T) {
	// Full happy path: CREATED → ... → ENROLLED
	transitions := []struct{ from, to State }{
		{StateCreated, StateIdentityVerified},
		{StateIdentityVerified, StateIntakeReady},
		{StateIntakeReady, StateIntakeInProgress},
		{StateIntakeInProgress, StateIntakeCompleted},
		{StateIntakeCompleted, StateEnrolled},
	}
	for _, tt := range transitions {
		if !CanTransition(tt.from, tt.to) {
			t.Errorf("expected valid transition %s → %s", tt.from, tt.to)
		}
	}
}

func TestCanTransition_HardStop(t *testing.T) {
	if !CanTransition(StateIntakeInProgress, StateHardStopped) {
		t.Error("IN_PROGRESS → HARD_STOPPED should be valid")
	}
	// HARD_STOPPED is terminal
	if CanTransition(StateHardStopped, StateIntakeInProgress) {
		t.Error("HARD_STOPPED → IN_PROGRESS should be invalid")
	}
}

func TestCanTransition_PauseResume(t *testing.T) {
	if !CanTransition(StateIntakeInProgress, StateIntakePaused) {
		t.Error("IN_PROGRESS → PAUSED should be valid")
	}
	if !CanTransition(StateIntakePaused, StateIntakeInProgress) {
		t.Error("PAUSED → IN_PROGRESS should be valid (resume)")
	}
}

func TestCanTransition_InvalidSkip(t *testing.T) {
	if CanTransition(StateCreated, StateIntakeInProgress) {
		t.Error("CREATED → IN_PROGRESS should be invalid (skips verification)")
	}
	if CanTransition(StateEnrolled, StateCreated) {
		t.Error("ENROLLED → CREATED should be invalid (terminal)")
	}
}

func TestEnrollment_Transition(t *testing.T) {
	e := &Enrollment{
		PatientID:   uuid.New(),
		TenantID:    uuid.New(),
		ChannelType: ChannelCorporate,
		State:       StateCreated,
		EncounterID: uuid.New(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := e.Transition(StateIdentityVerified); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}
	if e.State != StateIdentityVerified {
		t.Errorf("expected IDENTITY_VERIFIED, got %s", e.State)
	}

	// Invalid transition
	err := e.Transition(StateEnrolled)
	if err == nil {
		t.Fatal("expected error for invalid transition")
	}
	if _, ok := err.(*ErrInvalidTransition); !ok {
		t.Errorf("expected ErrInvalidTransition, got %T", err)
	}
}

func TestEnrollment_IsTerminal(t *testing.T) {
	e := &Enrollment{State: StateHardStopped}
	if !e.IsTerminal() {
		t.Error("HARD_STOPPED should be terminal")
	}

	e.State = StateEnrolled
	if !e.IsTerminal() {
		t.Error("ENROLLED should be terminal")
	}

	e.State = StateIntakeInProgress
	if e.IsTerminal() {
		t.Error("IN_PROGRESS should not be terminal")
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/enrollment/... -v -count=1`
Expected: All 7 tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/enrollment/
git commit -m "feat(intake): add 8-state enrollment state machine

States: CREATED → IDENTITY_VERIFIED → INTAKE_READY → INTAKE_IN_PROGRESS →
{HARD_STOPPED|INTAKE_PAUSED|INTAKE_COMPLETED} → ENROLLED.
3 channel types: CORPORATE, INSURANCE, GOVERNMENT.
Transition validation with ErrInvalidTransition error type."
```

---

## Task 6: PostgreSQL Migrations

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/migrations/001_init.sql`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/migrations/001_init.sql`

- [ ] **Step 1: Write ingestion migration**

Copy SQL from spec section 7.7 "Ingestion Service" — `lab_code_mappings`, `dlq_messages`, `patient_pending_queue` with indexes.

- [ ] **Step 2: Write intake migration**

Copy SQL from spec section 7.7 "Intake-Onboarding Service" — `enrollments`, `slot_events`, `current_slots` view, `flow_positions`, `review_queue` with indexes.

- [ ] **Step 3: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/migrations/ \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/migrations/
git commit -m "feat: add PostgreSQL migrations for ingestion and intake services

Ingestion: lab_code_mappings, dlq_messages, patient_pending_queue.
Intake: enrollments, slot_events (event-sourced), current_slots view,
flow_positions, review_queue."
```

---

## Task 7: API Gateway Updates

**Files:**
- Modify: `backend/services/api-gateway/app/config.py:7-24`
- Modify: `backend/services/api-gateway/app/api/proxy.py:101-108` (replace device_ingestion)
- Modify: `backend/services/api-gateway/app/api/proxy.py:302-376` (add permission branches)
- Modify: `backend/services/api-gateway/app/middleware/rbac.py:13-104` (add ROUTE_PERMISSIONS + ROLE_ROUTE_RESTRICTIONS)

- [ ] **Step 1: Add config settings**

In `backend/services/api-gateway/app/config.py`, add after line 21 (`WORKFLOW_ENGINE_SERVICE_URL`):

```python
    # Vaidshala Clinical Runtime Services
    INGESTION_SERVICE_URL: str = os.getenv("INGESTION_SERVICE_URL", "http://localhost:8140")
    INTAKE_SERVICE_URL: str = os.getenv("INTAKE_SERVICE_URL", "http://localhost:8141")
```

- [ ] **Step 2: Replace device_ingestion route in proxy.py**

Replace lines 101-108 (the `"device_ingestion"` entry) with these 4 entries. **ORDER MATTERS** — specific prefixes before broader ones:

```python
    # Ingestion Service — FHIR inbound (must be before broader /ingest prefix)
    "ingestion_fhir": {
        "prefix": "/api/v1/ingest/fhir",
        "target": settings.INGESTION_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Ingestion Service — source-specific receivers
    "ingestion": {
        "prefix": "/api/v1/ingest",
        "target": settings.INGESTION_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Intake-Onboarding Service — FHIR CRUD + $operations (must be before broader /intake prefix)
    "intake_fhir": {
        "prefix": "/api/v1/intake/fhir",
        "target": settings.INTAKE_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
    # Intake-Onboarding Service
    "intake_onboarding": {
        "prefix": "/api/v1/intake",
        "target": settings.INTAKE_SERVICE_URL,
        "strip_prefix": False,
        "public_paths": []
    },
```

- [ ] **Step 3: Update check_permissions_with_auth_service() in proxy.py**

Add two new `elif` blocks before the final catch-all deny (before line 374). The intake block must be granular to prevent patients from accessing review-only endpoints:

```python
    elif path.startswith("/api/v1/intake"):
        # Intake — review endpoints (pharmacist/physician only)
        if "$approve" in path or "$escalate" in path or "$request-clarification" in path:
            if request.method == "POST" and "intake:review" in user_permissions:
                return True, status.HTTP_200_OK, ""
        # Intake — safety alerts (pharmacist/physician)
        elif "DetectedIssue" in path:
            if request.method == "GET" and "safety:read" in user_permissions:
                return True, status.HTTP_200_OK, ""
        # Intake — enrollment
        elif "$enroll" in path or "$verify" in path or "$link-abha" in path:
            if request.method == "POST" and "intake:enroll" in user_permissions:
                return True, status.HTTP_200_OK, ""
        # Intake — check-in
        elif "$checkin" in path:
            if request.method == "POST" and "intake:checkin" in user_permissions:
                return True, status.HTTP_200_OK, ""
        # Intake — general read/write (slots, observations)
        else:
            if request.method == "GET" and "intake:read" in user_permissions:
                return True, status.HTTP_200_OK, ""
            elif request.method in ["POST", "PUT"] and "intake:write" in user_permissions:
                return True, status.HTTP_200_OK, ""
    elif path.startswith("/api/v1/ingest"):
        # Ingestion — admin endpoints
        if "$source-status" in path or "OperationOutcome" in path:
            if request.method == "GET" and "ingest:admin" in user_permissions:
                return True, status.HTTP_200_OK, ""
        # Ingestion — lab/EHR/ABDM webhooks (system role)
        elif "/labs" in path and request.method == "POST" and "ingest:lab" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif "/ehr" in path and request.method == "POST" and "ingest:ehr" in user_permissions:
            return True, status.HTTP_200_OK, ""
        elif "/abdm" in path and request.method == "POST" and "ingest:abdm" in user_permissions:
            return True, status.HTTP_200_OK, ""
        # Ingestion — device/wearable
        elif ("/devices" in path or "/wearables" in path) and request.method == "POST" and "ingest:device" in user_permissions:
            return True, status.HTTP_200_OK, ""
        # Ingestion — general observation write
        elif request.method == "POST" and "ingest:write" in user_permissions:
            return True, status.HTTP_200_OK, ""
```

- [ ] **Step 4: Add ROUTE_PERMISSIONS to rbac.py**

Append to `ROUTE_PERMISSIONS` dict (before the closing `}`), all entries from spec section 4.3:

```python
    # === Intake-Onboarding Service ===
    # Patient self-service
    r"^/api/v1/intake/fhir/Patient/\$enroll": {
        "POST": ["intake:enroll"]
    },
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$verify": {
        "POST": ["intake:enroll"]
    },
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$checkin": {
        "POST": ["intake:checkin"]
    },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$fill-slot": {
        "POST": ["intake:write"]
    },
    r"^/api/v1/intake/fhir/Observation": {
        "POST": ["intake:write"],
        "GET": ["intake:read"]
    },
    r"^/api/v1/intake/fhir/Patient": {
        "GET": ["patient:read"],
        "POST": ["patient:write"]
    },
    # Dashboard (Pharmacist + Physician)
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$approve": {
        "POST": ["intake:review"]
    },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$escalate": {
        "POST": ["intake:review"]
    },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$request-clarification": {
        "POST": ["intake:review"]
    },
    r"^/api/v1/intake/fhir/DetectedIssue": {
        "GET": ["safety:read"]
    },
    # === Ingestion Service ===
    r"^/api/v1/ingest/fhir/Observation": {
        "POST": ["ingest:write"]
    },
    r"^/api/v1/ingest/devices": {
        "POST": ["ingest:device"]
    },
    r"^/api/v1/ingest/wearables": {
        "POST": ["ingest:device"]
    },
    r"^/api/v1/ingest/fhir/OperationOutcome": {
        "GET": ["ingest:admin"]
    },
    r"^/api/v1/ingest/\$source-status": {
        "GET": ["ingest:admin"]
    },
    r"^/api/v1/ingest/labs": {
        "POST": ["ingest:lab"]
    },
    r"^/api/v1/ingest/ehr": {
        "POST": ["ingest:ehr"]
    },
    r"^/api/v1/ingest/abdm": {
        "POST": ["ingest:abdm"]
    },
```

- [ ] **Step 5: Add ROLE_ROUTE_RESTRICTIONS to rbac.py**

Append to `ROLE_ROUTE_RESTRICTIONS` dict:

```python
    # === Intake-Onboarding ===
    r"^/api/v1/intake/fhir/Patient/\$enroll":
        ["patient", "pharmacist", "physician", "asha"],
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$fill-slot":
        ["patient", "pharmacist", "physician", "asha"],
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$checkin":
        ["patient"],
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$(approve|escalate|request-clarification)":
        ["pharmacist", "physician"],
    r"^/api/v1/intake/fhir/DetectedIssue":
        ["pharmacist", "physician"],
    r"^/api/v1/intake/fhir/Encounter":
        ["pharmacist", "physician"],
    # === Ingestion ===
    r"^/api/v1/ingest/fhir/Observation":
        ["patient", "asha", "physician"],
    r"^/api/v1/ingest/devices":
        ["patient"],
    r"^/api/v1/ingest/wearables":
        ["patient"],
    r"^/api/v1/ingest/\$source-status":
        ["admin", "pharmacist", "physician"],
    r"^/api/v1/ingest/fhir/OperationOutcome":
        ["admin", "physician"],
    r"^/api/v1/ingest/labs":
        ["system", "physician"],
    r"^/api/v1/ingest/ehr":
        ["system", "physician"],
    r"^/api/v1/ingest/abdm":
        ["system"],
```

- [ ] **Step 6: Verify gateway starts**

Run: `cd backend/services/api-gateway && python -c "from app.config import settings; print(settings.INGESTION_SERVICE_URL, settings.INTAKE_SERVICE_URL)"`
Expected: `http://localhost:8140 http://localhost:8141`

- [ ] **Step 7: Commit**

```bash
git add backend/services/api-gateway/app/config.py \
       backend/services/api-gateway/app/api/proxy.py \
       backend/services/api-gateway/app/middleware/rbac.py
git commit -m "feat(gateway): add ingestion and intake service routes with RBAC

Replace device_ingestion with 4 new routes (order matters for startswith
matching). Update check_permissions_with_auth_service() with intake/ingest
elif branches. Add 18 ROUTE_PERMISSIONS + 14 ROLE_ROUTE_RESTRICTIONS."
```

---

## Task 8: Makefiles

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/Makefile`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/Makefile`

- [ ] **Step 1: Write ingestion Makefile**

```makefile
# Ingestion Service — Port 8140
.PHONY: help build run test test-cover clean docker lint fmt health

help:
	@echo "Ingestion Service — Port 8140"
	@echo "  build       Build the binary"
	@echo "  run         Build and run"
	@echo "  test        Run all tests"
	@echo "  test-cover  Run tests with coverage"
	@echo "  clean       Clean build artifacts"
	@echo "  docker      Build Docker image"
	@echo "  lint        Run linter"
	@echo "  fmt         Format code"
	@echo "  health      Check service health"

build:
	@go build -o bin/ingestion-service ./cmd/ingestion
	@echo "Built: bin/ingestion-service"

run: build
	@./bin/ingestion-service

test:
	@go test -v ./...

test-cover:
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean:
	@rm -rf bin/ coverage.out coverage.html

docker:
	@cd ../.. && docker build -f services/ingestion-service/Dockerfile -t ingestion-service:latest .

lint:
	@golangci-lint run ./...

fmt:
	@go fmt ./...

health:
	@curl -s http://localhost:8140/healthz | python3 -m json.tool
```

- [ ] **Step 2: Write intake Makefile**

Same structure as ingestion Makefile but with:
- Port 8141 in help text and health target
- Binary name `intake-onboarding-service`
- Build path `./cmd/intake`
- Docker target: `@cd ../.. && docker build -f services/intake-onboarding-service/Dockerfile -t intake-onboarding-service:latest .`

- [ ] **Step 3: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/Makefile \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/Makefile
git commit -m "feat: add Makefiles for ingestion and intake services"
```

---

## Task 9: Dockerfiles

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/Dockerfile`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/Dockerfile`

- [ ] **Step 1: Write ingestion Dockerfile**

```dockerfile
# Ingestion Service — Multi-stage build
FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates

# Copy parent module for replace directive
COPY go.mod go.sum ./
COPY pkg/ pkg/

# Copy service source
COPY services/ingestion-service/ services/ingestion-service/

WORKDIR /app/services/ingestion-service
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /ingestion-service ./cmd/ingestion

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -g '' appuser
COPY --from=builder /ingestion-service /app/ingestion-service
USER appuser
EXPOSE 8140

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8140/healthz || exit 1

CMD ["/app/ingestion-service"]
```

**Note:** The Dockerfile COPY context must be the `clinical-runtime-platform/` root (not the service dir) so the `replace ../../` directive resolves. Build with: `docker build -f services/ingestion-service/Dockerfile .` from the platform root.

- [ ] **Step 2: Write intake Dockerfile**

Same structure as ingestion Dockerfile but with:
- Binary name `intake-onboarding-service`, build path `./cmd/intake`
- Port 8141 (EXPOSE and HEALTHCHECK)
- Service directory: `services/intake-onboarding-service/`

- [ ] **Step 3: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/Dockerfile \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/Dockerfile
git commit -m "feat: add multi-stage Dockerfiles for ingestion and intake services

Build context is clinical-runtime-platform/ root for replace directive.
Alpine-based, non-root user, healthcheck on /healthz."
```

---

## Task 10: Verify End-to-End

- [ ] **Step 1: Compile both services**

```bash
cd vaidshala/clinical-runtime-platform/services/ingestion-service && go build ./cmd/ingestion/ && echo "Ingestion OK"
cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go build ./cmd/intake/ && echo "Intake OK"
```

- [ ] **Step 2: Run all tests**

```bash
cd vaidshala/clinical-runtime-platform && go test ./pkg/fhirclient/... -v -count=1
cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./... -v -count=1
cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./... -v -count=1
```

- [ ] **Step 3: Boot ingestion service (smoke test)**

```bash
cd vaidshala/clinical-runtime-platform/services/ingestion-service
FHIR_ENABLED=false DATABASE_URL=postgres://postgres:postgres@localhost:5433/ingestion_service?sslmode=disable go run ./cmd/ingestion/ &
sleep 2
curl -s http://localhost:8140/healthz | python3 -m json.tool
# Expected: {"service": "ingestion-service", "status": "ok"}
kill %1
```

- [ ] **Step 4: Boot intake service (smoke test)**

```bash
cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service
FHIR_ENABLED=false DATABASE_URL=postgres://postgres:postgres@localhost:5433/intake_service?sslmode=disable go run ./cmd/intake/ &
sleep 2
curl -s http://localhost:8141/healthz | python3 -m json.tool
# Expected: {"service": "intake-onboarding-service", "status": "ok"}
kill %1
```

- [ ] **Step 5: Verify gateway config**

```bash
cd backend/services/api-gateway
python3 -c "
from app.config import settings
assert settings.INGESTION_SERVICE_URL == 'http://localhost:8140'
assert settings.INTAKE_SERVICE_URL == 'http://localhost:8141'
print('Gateway config OK')
"
```

- [ ] **Step 6: Final commit (if any fixups needed)**

```bash
git add -A
git status
# Only commit if there are actual changes
```

---

## Summary

| Task | Deliverable | Tests |
|------|------------|-------|
| 1 | `pkg/fhirclient` — shared Google FHIR Store client | 6 unit tests |
| 2 | Ingestion service scaffolding (compile + health) | Compile check |
| 3 | Intake service scaffolding (compile + health) | Compile check |
| 4 | Canonical types + pipeline interfaces | 3 unit tests |
| 5 | Enrollment state machine (8 states, transitions) | 7 unit tests |
| 6 | PostgreSQL migrations (both services) | SQL review |
| 7 | API Gateway updates (routes + RBAC) | Config verify |
| 8 | Makefiles | — |
| 9 | Dockerfiles | — |
| 10 | End-to-end verification | Smoke tests |

**Remaining plans (separate documents):**
- **Plan 2:** `2026-03-21-ingestion-core.md` — Pipeline stages, normalizer, validator, FHIR mapper, Kafka producer, patient-reported + device adapters
- **Plan 3:** `2026-03-21-intake-core.md` — Slot table (50 slots), safety engine (11 HARD_STOPs, 8 SOFT_FLAGs), flow graph engine, Flutter app handler, FHIR writes
- **Plan 4:** `2026-03-21-channels-integration.md` — WhatsApp adapter, ASHA tablet, ABDM HIU/HIP, lab adapters (6 labs), EHR adapter (HL7v2/FHIR/SFTP)
- **Plan 5:** `2026-03-21-advanced-features.md` — Biweekly check-in (M0-CI), pharmacist review queue, wearable adapters, DLQ management, full observability
